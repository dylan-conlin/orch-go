package events

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
)

// SeekToTimestamp estimates the byte offset in an events.jsonl file where events
// around the target unix timestamp begin. It reads the first and last timestamps,
// interpolates a file position, seeks there, and skips to the next complete line.
// Returns a reader positioned after the seek, or (nil, false) if seeking isn't beneficial.
func SeekToTimestamp(file *os.File, since int64) (io.Reader, bool) {
	stat, err := file.Stat()
	if err != nil || stat.Size() < 4096 {
		return nil, false // too small to bother seeking
	}
	fileSize := stat.Size()

	// Read first timestamp
	firstTS := readFirstTimestamp(file)
	if firstTS == 0 {
		file.Seek(0, io.SeekStart)
		return nil, false
	}

	// Read last timestamp
	lastTS := readLastTimestamp(file, fileSize)
	if lastTS == 0 || lastTS <= firstTS {
		file.Seek(0, io.SeekStart)
		return nil, false
	}

	// If since is before the first event, read everything
	if since <= firstTS {
		file.Seek(0, io.SeekStart)
		return nil, false
	}

	// Interpolate: what fraction of the file should we skip?
	totalDuration := float64(lastTS - firstTS)
	skipDuration := float64(since - firstTS)
	skipFraction := skipDuration / totalDuration

	// Apply a safety margin — seek to 20% earlier than estimated
	seekFraction := skipFraction * 0.8
	if seekFraction <= 0 {
		file.Seek(0, io.SeekStart)
		return nil, false
	}

	seekPos := int64(float64(fileSize) * seekFraction)
	file.Seek(seekPos, io.SeekStart)

	// Skip the partial line at seek position
	br := bufio.NewReader(file)
	_, err = br.ReadBytes('\n')
	if err != nil {
		// If we can't find a newline, fall back to start
		file.Seek(0, io.SeekStart)
		return nil, false
	}

	return br, true
}

// readFirstTimestamp reads the first valid timestamp from the beginning of the file.
func readFirstTimestamp(file *os.File) int64 {
	file.Seek(0, io.SeekStart)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var event struct {
			Timestamp int64 `json:"timestamp"`
		}
		if err := json.Unmarshal([]byte(line), &event); err == nil && event.Timestamp > 0 {
			return event.Timestamp
		}
	}
	return 0
}

// readLastTimestamp reads the last valid timestamp from the end of the file.
func readLastTimestamp(file *os.File, fileSize int64) int64 {
	// Read the last 4KB to find the last line
	readSize := int64(4096)
	if readSize > fileSize {
		readSize = fileSize
	}
	file.Seek(fileSize-readSize, io.SeekStart)

	data := make([]byte, readSize)
	n, err := io.ReadFull(file, data)
	if err != nil && err != io.ErrUnexpectedEOF {
		return 0
	}
	data = data[:n]

	// Scan backwards for the last complete line
	lastNewline := -1
	for i := len(data) - 1; i >= 0; i-- {
		if data[i] == '\n' {
			if lastNewline == -1 {
				lastNewline = i
				continue
			}
			// Found the start of the last line
			line := data[i+1 : lastNewline]
			var event struct {
				Timestamp int64 `json:"timestamp"`
			}
			if err := json.Unmarshal(line, &event); err == nil && event.Timestamp > 0 {
				return event.Timestamp
			}
			break
		}
	}

	// Edge case: only one line in the tail chunk
	if lastNewline >= 0 {
		line := data[:lastNewline]
		var event struct {
			Timestamp int64 `json:"timestamp"`
		}
		if err := json.Unmarshal(line, &event); err == nil && event.Timestamp > 0 {
			return event.Timestamp
		}
	}

	return 0
}
