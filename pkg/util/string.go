package util

import (
	"strings"
	"unicode"
)

// IsDigit -
func IsDigit(str string) bool {
	for _, r := range str {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// IsContainItem -
func IsContainItem(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

// FormatTime2String -
func FormatTime2String(s string) string {
	s = strings.Replace(s, "-", "", -1)
	s = strings.Replace(s, " ", "", -1)
	s = strings.Replace(s, ":", "", -1)
	return s
}
