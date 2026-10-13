package domain

import (
	"net/url"
	"strings"
)

// SourceAudioFile permits safe nested source paths, never local preservation paths.
func SourceAudioFile(file string) bool {
	if strings.ContainsAny(file, "\\\x00") {
		return false
	}
	for _, char := range file {
		if char < 32 || char == 127 {
			return false
		}
	}
	for _, part := range strings.Split(file, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func SourceAudioPath(file string) string {
	parts := strings.Split(file, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
