package questionbank

import (
	"errors"
	"exam-paper/internal/repository"
	"exam-paper/internal/model"
	"exam-paper/internal/service/parser"
	"exam-paper/internal/utils"
	"os"
	"path/filepath"
	"sync"
)

type Service struct {
	rootDir string
	store   *repository.Store
	cache   struct {
		sync.Mutex
		bySource map[string][]model.Question
		byID     map[string]model.Question
	}
}

func New(rootDir string, store *repository.Store) *Service {
	s := &Service{rootDir: rootDir, store: store}
	s.cache.bySource = map[string][]model.Question{}
	s.cache.byID = map[string]model.Question{}
	return s
}

func (s *Service) RememberQuestions(source string, questions []model.Question) {
	source = utils.CanonicalSource(source)
	s.cache.Lock()
	defer s.cache.Unlock()
	s.cache.bySource[source] = append([]model.Question(nil), questions...)
	for _, q := range questions {
		s.cache.byID[q.ID] = q
	}
}

func (s *Service) ExamPayloadFor(exam model.ExamRecord) (model.ExamPayload, error) {
	questions := make([]model.Question, 0, len(exam.QuestionIDs))
	for _, id := range exam.QuestionIDs {
		q, ok := s.QuestionByID(id)
		if !ok {
			return model.ExamPayload{}, errors.New("试卷中的题目来源已丢失。")
		}
		questions = append(questions, q)
	}
	if exam.Answers == nil {
		exam.Answers = map[string]model.AnswerRecord{}
	}
	var doubts []string
	if s.store != nil {
		doubts = s.store.DoubtIDs(exam.UserID, exam.QuestionIDs)
	}
	return model.ExamPayload{Exam: exam, Questions: questions, Answers: exam.Answers, Doubts: doubts}, nil
}

func (s *Service) QuestionsBySources(sources []string) ([]model.Question, error) {
	var all []model.Question
	for _, source := range utils.CleanSources(sources) {
		questions, err := s.QuestionsForSource(source)
		if err != nil {
			return nil, err
		}
		all = append(all, questions...)
	}
	return all, nil
}

func (s *Service) QuestionsForSource(source string) ([]model.Question, error) {
	source = utils.CanonicalSource(source)
	s.cache.Lock()
	cached, ok := s.cache.bySource[source]
	s.cache.Unlock()
	if ok {
		return append([]model.Question(nil), cached...), nil
	}
	if s.store != nil {
		questions, ok, err := s.store.QuestionsBySource(source)
		if err != nil {
			return nil, err
		}
		if ok {
			s.RememberQuestions(source, questions)
			return append([]model.Question(nil), questions...), nil
		}
	}

	target := filepath.Join(s.rootDir, filepath.Clean(source))
	if !utils.Within(target, s.rootDir) {
		return nil, errors.New("文件路径不在工作目录内。")
	}
	result, err := parser.ParseQuestionFile(target, source)
	if err != nil {
		return nil, err
	}
	s.RememberQuestions(source, result.Questions)
	if s.store != nil {
		if info, statErr := os.Stat(target); statErr == nil {
			_ = s.store.ReplaceSourceQuestions(source, info, result.Questions)
		}
	}
	return append([]model.Question(nil), result.Questions...), nil
}

func (s *Service) QuestionByID(id string) (model.Question, bool) {
	s.cache.Lock()
	q, ok := s.cache.byID[id]
	s.cache.Unlock()
	if ok {
		return q, true
	}
	if s.store != nil {
		q, ok, err := s.store.QuestionByID(id)
		if err == nil && ok {
			s.cache.Lock()
			s.cache.byID[q.ID] = q
			s.cache.Unlock()
			return q, true
		}
	}
	return model.Question{}, false
}

func (s *Service) QuestionSource(questionID string) string {
	if q, ok := s.QuestionByID(questionID); ok {
		return q.Source
	}
	return ""
}



