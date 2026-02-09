package events

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	// MaxEventsLogSizeBytes is the max size of events.jsonl before rotation.
	MaxEventsLogSizeBytes int64 = 5 * 1024 * 1024
	// MaxRotatedEventLogs is the number of rotated archives to keep.
	MaxRotatedEventLogs = 3

	jsonlScannerMaxBytes = 1024 * 1024
)

func maybeRotateLog(path string, incomingBytes int64) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if info.Size()+incomingBytes <= MaxEventsLogSizeBytes {
		return nil
	}

	return rotateLogFiles(path, MaxRotatedEventLogs)
}

func rotateLogFiles(path string, maxRotated int) error {
	if maxRotated < 1 {
		return nil
	}

	oldestPath := fmt.Sprintf("%s.%d", path, maxRotated)
	if err := os.Remove(oldestPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	for i := maxRotated - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", path, i)
		dst := fmt.Sprintf("%s.%d", path, i+1)

		if _, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}

		if err := os.Rename(src, dst); err != nil {
			return err
		}
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return os.Rename(path, path+".1")
}

// CompactedLogPaths returns existing log paths in chronological order
// (oldest rotated archive first, current events.jsonl last).
func CompactedLogPaths(path string) ([]string, error) {
	paths := make([]string, 0, MaxRotatedEventLogs+1)

	for i := MaxRotatedEventLogs; i >= 1; i-- {
		rotated := fmt.Sprintf("%s.%d", path, i)
		exists, err := pathExists(rotated)
		if err != nil {
			return nil, err
		}
		if exists {
			paths = append(paths, rotated)
		}
	}

	exists, err := pathExists(path)
	if err != nil {
		return nil, err
	}
	if exists {
		paths = append(paths, path)
	}

	if len(paths) == 0 {
		return nil, os.ErrNotExist
	}

	return paths, nil
}

// ReadCompactedJSONL reads JSONL lines from rotated archives + current log
// in chronological order.
func ReadCompactedJSONL(path string, handleLine func(line string) error) error {
	paths, err := CompactedLogPaths(path)
	if err != nil {
		return err
	}

	for _, currentPath := range paths {
		file, err := os.Open(currentPath)
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 0, 64*1024), jsonlScannerMaxBytes)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			if err := handleLine(line); err != nil {
				_ = file.Close()
				return err
			}
		}

		if err := scanner.Err(); err != nil {
			_ = file.Close()
			return err
		}

		if err := file.Close(); err != nil {
			return err
		}
	}

	return nil
}

func pathExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
