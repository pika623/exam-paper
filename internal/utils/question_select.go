package utils

import (
	"crypto/sha1"
	"encoding/hex"
	"exam-paper/internal/model"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var FileTypes = map[string]bool{".pdf": true, ".docx": true}

func PreferWrongThenShuffle(wrongCounts map[string]int, questions []model.Question) {
	seed := timeSeed()
	sort.SliceStable(questions, func(i, j int) bool {
		wi := wrongCounts[questions[i].ID]
		wj := wrongCounts[questions[j].ID]
		if wi == wj {
			return hashRank(questions[i].ID, seed) < hashRank(questions[j].ID, seed)
		}
		return wi > wj
	})
}

func CleanSources(sources []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, source := range sources {
		source = CanonicalSource(source)
		if source == "" || seen[source] {
			continue
		}
		seen[source] = true
		out = append(out, source)
	}
	return out
}

func CanonicalSource(source string) string {
	return filepath.ToSlash(strings.TrimSpace(source))
}

func Within(path string, base string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absBase, absPath)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func timeSeed() string {
	return time.Now().Format(time.RFC3339Nano)
}

func hashRank(value string, seed string) string {
	sum := sha1.Sum([]byte(value + ":" + seed))
	return hex.EncodeToString(sum[:])
}


