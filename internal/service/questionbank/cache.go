package questionbank

import (
	"errors"
	"exam-paper/internal/model"
	"exam-paper/internal/repository"
	"exam-paper/internal/service/parser"
	"exam-paper/internal/utils"
	"os"
	"path/filepath"
	"sync"
)

const DefaultExamQuestionWindow = 25

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
	if exam.Answers == nil {
		exam.Answers = map[string]model.AnswerRecord{}
	}
	var doubts []string
	if s.store != nil {
		doubts = s.store.DoubtIDs(exam.UserID, exam.QuestionIDs)
	}
	questions, offset, err := s.ExamQuestionWindow(exam, exam.Current, DefaultExamQuestionWindow)
	if err != nil {
		return model.ExamPayload{}, err
	}
	return model.ExamPayload{
		Exam:      exam,
		Questions: questions,
		Offset:    offset,
		Total:     len(exam.QuestionIDs),
		Answers:   exam.Answers,
		Doubts:    doubts,
	}, nil
}

func (s *Service) ExamQuestionsPayload(exam model.ExamRecord, center int, limit int) (model.ExamQuestionsPayload, error) {
	questions, offset, err := s.ExamQuestionWindow(exam, center, limit)
	if err != nil {
		return model.ExamQuestionsPayload{}, err
	}
	return model.ExamQuestionsPayload{Questions: questions, Offset: offset, Total: len(exam.QuestionIDs)}, nil
}

func (s *Service) ExamQuestionWindow(exam model.ExamRecord, center int, limit int) ([]model.Question, int, error) {
	total := len(exam.QuestionIDs)
	if total == 0 {
		return nil, 0, nil
	}
	if limit <= 0 {
		limit = DefaultExamQuestionWindow
	}
	if limit > total {
		limit = total
	}
	if center < 0 {
		center = 0
	}
	if center >= total {
		center = total - 1
	}
	offset := center - limit/2
	if offset < 0 {
		offset = 0
	}
	if offset+limit > total {
		offset = total - limit
	}
	questions := make([]model.Question, 0, limit)
	for _, id := range exam.QuestionIDs[offset : offset+limit] {
		q, ok := s.QuestionByID(id)
		if !ok {
			return nil, 0, errors.New("试卷中的题目来源已丢失。")
		}
		q.Raw = ""
		questions = append(questions, q)
	}
	return questions, offset, nil
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
