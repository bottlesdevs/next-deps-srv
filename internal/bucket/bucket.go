package bucket

import (
	"strings"
)

func Char(filename string) string {
	if len(filename) == 0 {
		return "-"
	}
	c := strings.ToLower(string([]rune(filename)[0]))
	if (c >= "a" && c <= "z") || (c >= "0" && c <= "9") {
		return c
	}
	return "-"
}

func BucketPath(root, filename string) string {
	return root + "/" + Char(filename) + "/" + filename
}

var bucketChars []string

func init() {
	for c := 'a'; c <= 'z'; c++ {
		bucketChars = append(bucketChars, string(c))
	}
	for c := '0'; c <= '9'; c++ {
		bucketChars = append(bucketChars, string(c))
	}
	bucketChars = append(bucketChars, "-")
}

func AllChars() []string { return bucketChars }
