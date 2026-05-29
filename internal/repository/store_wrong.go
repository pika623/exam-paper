package repository

import (
	"encoding/json"
	"exam-paper/internal/model"
)

func (s *Store) ExamWrongQuestions(userID string, examID string) ([]model.WrongQuestion, error) {
	page, err := s.ExamWrongQuestionPage(userID, examID, 0, 0)
	return page.Items, err
}

func (s *Store) ExamWrongQuestionPage(userID string, examID string, offset int, limit int) (model.WrongQuestionPage, error) {
	exam, err := s.ExamByID(userID, examID)
	if err != nil {
		return model.WrongQuestionPage{}, err
	}
	offset, limit = normalizePage(offset, limit)
	if limit <= 0 {
		limit = len(exam.QuestionIDs)
	}
	var (
		wrongs []model.WrongQuestion
		total  int
	)
	for _, questionID := range exam.QuestionIDs {
		answer, ok := exam.Answers[questionID]
		if !ok || !answer.Judged || answer.Correct {
			continue
		}
		if total < offset || len(wrongs) >= limit {
			total++
			continue
		}
		q, ok := s.questionByID(questionID)
		if !ok {
			total++
			continue
		}
		q.Raw = ""
		wrongs = append(wrongs, model.WrongQuestion{
			Question: q,
			Answer:   answer,
			Wrong:    s.wrongRecord(userID, questionID),
		})
		total++
	}
	return wrongQuestionPage(wrongs, total, offset, limit), nil
}

func (s *Store) WrongBook(userID string) ([]model.WrongQuestion, error) {
	page, err := s.WrongBookPage(userID, 0, 0)
	return page.Items, err
}

func (s *Store) WrongBookPage(userID string, offset int, limit int) (model.WrongQuestionPage, error) {
	offset, limit = normalizePage(offset, limit)
	total, err := s.wrongBookTotal(userID, "")
	if err != nil {
		return model.WrongQuestionPage{}, err
	}
	if limit <= 0 {
		limit = total
	}
	rows, err := s.db.Query(
		`SELECT question_id, source, wrong_count, correct_count, last_wrong_at, last_selected_json
		 FROM wrong_book WHERE user_id = ? ORDER BY wrong_count DESC, last_wrong_at DESC LIMIT ? OFFSET ?`, userID, limit, offset,
	)
	if err != nil {
		return model.WrongQuestionPage{}, err
	}

	var records []model.WrongRecord
	for rows.Next() {
		var record model.WrongRecord
		var selectedJSON string
		if err := rows.Scan(&record.QuestionID, &record.Source, &record.WrongCount, &record.CorrectCount, &record.LastWrongAt, &selectedJSON); err != nil {
			rows.Close()
			return model.WrongQuestionPage{}, err
		}
		_ = json.Unmarshal([]byte(selectedJSON), &record.LastSelected)
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return model.WrongQuestionPage{}, err
	}
	rows.Close()

	var wrongs []model.WrongQuestion
	for _, record := range records {
		q, ok := s.questionByID(record.QuestionID)
		if !ok {
			continue
		}
		q.Raw = ""
		wrongs = append(wrongs, model.WrongQuestion{
			Question: q,
			Wrong:    record,
			Answer:   model.AnswerRecord{Selected: record.LastSelected, Correct: false, Judged: true, AnsweredAt: record.LastWrongAt},
		})
	}
	return wrongQuestionPage(wrongs, total, offset, limit), nil
}

func (s *Store) wrongRecord(userID string, questionID string) model.WrongRecord {
	var record model.WrongRecord
	var selectedJSON string
	_ = s.db.QueryRow(
		`SELECT question_id, source, wrong_count, correct_count, last_wrong_at, last_selected_json FROM wrong_book WHERE user_id = ? AND question_id = ?`,
		userID, questionID,
	).Scan(&record.QuestionID, &record.Source, &record.WrongCount, &record.CorrectCount, &record.LastWrongAt, &selectedJSON)
	_ = json.Unmarshal([]byte(selectedJSON), &record.LastSelected)
	return record
}

func (s *Store) FeaturedWrongBook(userID string) ([]model.WrongQuestion, error) {
	page, err := s.FeaturedWrongBookPage(userID, 0, 0)
	return page.Items, err
}

func (s *Store) FeaturedWrongBookPage(userID string, offset int, limit int) (model.WrongQuestionPage, error) {
	offset, limit = normalizePage(offset, limit)
	total, err := s.wrongBookTotal(userID, "featured")
	if err != nil {
		return model.WrongQuestionPage{}, err
	}
	if limit <= 0 {
		limit = total
	}
	rows, err := s.db.Query(
		`SELECT question_id, source, wrong_count, correct_count, last_wrong_at, last_selected_json
		 FROM wrong_book
		 WHERE user_id = ? AND wrong_count - correct_count >= -3
		 ORDER BY (wrong_count - correct_count) DESC, wrong_count DESC, last_wrong_at DESC LIMIT ? OFFSET ?`, userID, limit, offset,
	)
	if err != nil {
		return model.WrongQuestionPage{}, err
	}

	var records []model.WrongRecord
	for rows.Next() {
		var record model.WrongRecord
		var selectedJSON string
		if err := rows.Scan(&record.QuestionID, &record.Source, &record.WrongCount, &record.CorrectCount, &record.LastWrongAt, &selectedJSON); err != nil {
			rows.Close()
			return model.WrongQuestionPage{}, err
		}
		_ = json.Unmarshal([]byte(selectedJSON), &record.LastSelected)
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return model.WrongQuestionPage{}, err
	}
	rows.Close()

	var wrongs []model.WrongQuestion
	for _, record := range records {
		q, ok := s.questionByID(record.QuestionID)
		if !ok {
			continue
		}
		q.Raw = ""
		wrongs = append(wrongs, model.WrongQuestion{
			Question: q,
			Wrong:    record,
			Answer:   model.AnswerRecord{Selected: record.LastSelected, Correct: false, Judged: true, AnsweredAt: record.LastWrongAt},
		})
	}
	return wrongQuestionPage(wrongs, total, offset, limit), nil
}

func normalizePage(offset int, limit int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if limit < 0 {
		limit = 0
	}
	return offset, limit
}

func wrongQuestionPage(items []model.WrongQuestion, total int, offset int, limit int) model.WrongQuestionPage {
	return model.WrongQuestionPage{
		Items:   items,
		Total:   total,
		Offset:  offset,
		Limit:   limit,
		HasMore: offset+len(items) < total,
	}
}

func (s *Store) wrongBookTotal(userID string, kind string) (int, error) {
	var (
		total int
		err   error
	)
	if kind == "featured" {
		err = s.db.QueryRow(`SELECT COUNT(*) FROM wrong_book WHERE user_id = ? AND wrong_count - correct_count >= -3`, userID).Scan(&total)
	} else {
		err = s.db.QueryRow(`SELECT COUNT(*) FROM wrong_book WHERE user_id = ?`, userID).Scan(&total)
	}
	return total, err
}

func (s *Store) WrongCounts(userID string) map[string]int {
	rows, err := s.db.Query(`SELECT question_id, wrong_count FROM wrong_book WHERE user_id = ?`, userID)
	if err != nil {
		return map[string]int{}
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var questionID string
		var count int
		if err := rows.Scan(&questionID, &count); err == nil {
			counts[questionID] = count
		}
	}
	return counts
}

func (s *Store) ExistingQuestionIDs(ids []string, limit int) []string {
	existing := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := s.questionByID(id); !ok {
			continue
		}
		existing = append(existing, id)
		if limit > 0 && len(existing) >= limit {
			break
		}
	}
	return existing
}

func (s *Store) questionByID(id string) (model.Question, bool) {
	if s.questions == nil {
		return model.Question{}, false
	}
	return s.questions.QuestionByID(id)
}

func (s *Store) questionSource(questionID string) string {
	if q, ok := s.questionByID(questionID); ok {
		return q.Source
	}
	return ""
}
