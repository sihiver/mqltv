package parser

import (
	"bufio"
	"io"
	"net/http"
	"strings"
)

type M3UChannel struct {
	Name  string
	URL   string
	Logo  string
	Group string
}

func ParseM3U(source io.Reader) ([]M3UChannel, error) {
	var channels []M3UChannel
	scanner := bufio.NewScanner(source)
	
	var currentChannel M3UChannel
	var hasInfo bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if strings.HasPrefix(line, "#EXTM3U") {
			continue
		}
		
		if strings.HasPrefix(line, "#EXTINF:") {
			hasInfo = true
			currentChannel = M3UChannel{}
			
			// Parse tvg-logo
			if logoStart := strings.Index(line, "tvg-logo=\""); logoStart != -1 {
				logoStart += 10
				logoEnd := strings.Index(line[logoStart:], "\"")
				if logoEnd != -1 {
					currentChannel.Logo = line[logoStart : logoStart+logoEnd]
				}
			}
			
			// Parse group-title
			if groupStart := strings.Index(line, "group-title=\""); groupStart != -1 {
				groupStart += 13
				groupEnd := strings.Index(line[groupStart:], "\"")
				if groupEnd != -1 {
					currentChannel.Group = line[groupStart : groupStart+groupEnd]
				}
			}
			
			// Parse channel name (after last comma)
			if commaIdx := strings.LastIndex(line, ","); commaIdx != -1 {
				currentChannel.Name = strings.TrimSpace(line[commaIdx+1:])
			}
		} else if hasInfo && line != "" && !strings.HasPrefix(line, "#") {
			currentChannel.URL = line
			channels = append(channels, currentChannel)
			hasInfo = false
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return channels, nil
}

func ParseM3UURL(url string) ([]M3UChannel, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ParseM3U(resp.Body)
}
