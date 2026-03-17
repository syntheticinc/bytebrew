package utils

// Truncate truncates a string to maxLen runes (not bytes) with ellipsis "..."
// This properly handles UTF-8 strings including Russian, Chinese, etc.
// If maxLen is 3 or less, the string is simply cut to maxLen without ellipsis.
func Truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}
