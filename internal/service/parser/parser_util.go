package parser

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
)

func questionID(source string, ordinal int, stem string, answer string) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s:%d:%s:%s", source, ordinal, stem, answer)))
	return hex.EncodeToString(sum[:])[:16]
}

func truncateRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit])
}


