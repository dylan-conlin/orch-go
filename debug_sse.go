//go:build ignore

package main

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
)

func main() {
	resp, err := http.Get("http://127.0.0.1:4096/event")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	var eventBuffer strings.Builder
	lineNum := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Read error: %v\n", err)
			break
		}
		lineNum++
		fmt.Printf("[%d] %q\n", lineNum, line)
		eventBuffer.WriteString(line)

		// Empty line signals end of event
		if line == "\n" && eventBuffer.Len() > 1 {
			raw := eventBuffer.String()
			fmt.Printf("--- EVENT RAW: %q\n", raw)
			// parse
			lines := strings.Split(raw, "\n")
			var eventType, data string
			for _, l := range lines {
				if strings.HasPrefix(l, "event: ") {
					eventType = strings.TrimPrefix(l, "event: ")
				} else if strings.HasPrefix(l, "data: ") {
					data = strings.TrimPrefix(l, "data: ")
				}
			}
			fmt.Printf("Parsed eventType=%q data=%q\n", eventType, data)
			eventBuffer.Reset()
		}
	}
}
