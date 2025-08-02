package main

import "strings"

func CropText(input string, length int) string {
	runes := []rune(input)
	if len(runes) <= length {
		return input
	}

	return string(runes[:length-3]) + "..."
}

func ExtractAfterLastThinkTag(input string) string {
	tag := "\n</think>\n\n"
	idx := strings.LastIndex(input, tag)
	if idx == -1 {
		return strings.TrimSpace(input)
	}
	return strings.TrimSpace(input[idx+len(tag):])
}
