package repository

import (
	"encoding/json"
	"exam-paper/internal/model"
)

func (s *Store) ExamWrongQuestions(userID string, examID string) ([]model.WrongQuestion, error) {
	exam, err := s.ExamByID(userID, examID)
	if err != nil {
		return nil, err
	}
	var wrongs []model.WrongQuestion
	for _, questionID := range exam.QuestionIDs {
		answer, ok := exam.Answers[questionID]
		if !ok || !answer.Judged || answer.Correct {
			continue
		}
		q, ok := s.questionByID(questionID)
		if !ok {
			continue
		}
		wrongs = append(wrongs, model.WrongQuestion{
			Question: q,
			Answer:   answer,
			Wrong:    s.wrongRecord(userID, questionID),
		})
	}
	return wrongs, nil
}

func (s *Store) WrongBook(userID string) ([]model.WrongQuestion, error) {
	rows, err := s.db.Query(
		`SELECT question_id, source, wrong_count, correct_count, last_wrong_at, last_selected_json
		 FROM wrong_book WHERE user_id = ? ORDER BY wrong_count DESC, last_wrong_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}

	var records []model.WrongRecord
	for rows.Next() {
		var record model.WrongRecord
		var selectedJSON string
		if err := rows.Scan(&record.QuestionID, &record.Source, &record.WrongCount, &record.CorrectCount, &record.LastWrongAt, &selectedJSON); err != nil {
			rows.Close()
			return nil, err
		}
		_ = json.Unmarshal([]byte(selectedJSON), &record.LastSelected)
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	var wrongs []model.WrongQuestion
	for _, record := range records {
		q, ok := s.questionByID(record.QuestionID)
		if !ok {
			continue
		}
		wrongs = append(wrongs, model.WrongQuestion{
			Question: q,
			Wrong:    record,
			Answer:   model.AnswerRecord{Selected: record.LastSelected, Correct: false, Judged: true, AnsweredAt: record.LastWrongAt},
		})
	}
	return wrongs, nil
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
	rows, err := s.db.Query(
		`SELECT question_id, source, wrong_count, correct_count, last_wrong_at, last_selected_json
		 FROM wrong_book
		 WHERE user_id = ? AND wrong_count - correct_count >= -3
		 ORDER BY (wrong_count - correct_count) DESC, wrong_count DESC, last_wrong_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}

	var records []model.WrongRecord
	for rows.Next() {
		var record model.WrongRecord
		var selectedJSON string
		if err := rows.Scan(&record.QuestionID, &record.Source, &record.WrongCount, &record.CorrectCount, &record.LastWrongAt, &selectedJSON); err != nil {
			rows.Close()
			return nil, err
		}
		_ = json.Unmarshal([]byte(selectedJSON), &record.LastSelected)
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	var wrongs []model.WrongQuestion
	for _, record := range records {
		q, ok := s.questionByID(record.QuestionID)
		if !ok {
			continue
		}
		wrongs = append(wrongs, model.WrongQuestion{
			Question: q,
			Wrong:    record,
			Answer:   model.AnswerRecord{Selected: record.LastSelected, Correct: false, Judged: true, AnsweredAt: record.LastWrongAt},
		})
	}
	return wrongs, nil
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


