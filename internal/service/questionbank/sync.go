package questionbank

import (
	"exam-paper/internal/service/parser"
	"exam-paper/internal/utils"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var bundledQuestionBankDirs = []string{
	"演出协会官方题库题目",
	"演出协会模拟考",
}

func (s *Service) SyncBundledQuestionBank() error {
	var total int
	for _, dir := range bundledQuestionBankDirs {
		count, err := s.syncBundledQuestionBankDir(dir)
		if err != nil {
			return err
		}
		total += count
	}
	log.Printf("synced bundled question banks: %d questions", total)
	return nil
}

func (s *Service) syncBundledQuestionBankDir(dir string) (int, error) {
	base := filepath.Join(s.rootDir, dir)
	info, err := os.Stat(base)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return 0, nil
	}

	var total int
	err = filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !utils.FileTypes[ext] {
			return nil
		}
		count, err := s.syncQuestionSource(path)
		if err != nil {
			return err
		}
		total += count
		return nil
	})
	if err != nil {
		return 0, err
	}
	log.Printf("synced bundled question bank %s: %d questions", dir, total)
	return total, nil
}

func (s *Service) syncQuestionSource(path string) (int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	rel, err := filepath.Rel(s.rootDir, path)
	if err != nil {
		return 0, err
	}
	source := filepath.ToSlash(rel)
	if s.store.SourceCurrent(source, info) {
		questions, ok, err := s.store.QuestionsBySource(source)
		if err != nil {
			return 0, err
		}
		if ok {
			s.RememberQuestions(source, questions)
			return len(questions), nil
		}
	}
	result, err := parser.ParseQuestionFile(path, source)
	if err != nil {
		return 0, err
	}
	if err := s.store.ReplaceSourceQuestions(source, info, result.Questions); err != nil {
		return 0, err
	}
	s.RememberQuestions(source, result.Questions)
	return len(result.Questions), nil
}


