package main

func CropText(input string) string {
	const maxLength = 4096

	runes := []rune(input)
	if len(runes) <= maxLength {
		return input
	}

	return string(runes[:maxLength-3]) + "..."
}
