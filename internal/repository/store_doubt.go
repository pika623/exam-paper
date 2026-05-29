package repository

import (
	"exam-paper/internal/model"
	"time"
)

func (s *Store) SetDoubt(userID string, questionID string, marked bool) error {
	if !marked {
		_, err := s.db.Exec(`DELETE FROM doubt_marks WHERE user_id = ? AND question_id = ?`, userID, questionID)
		return err
	}
	source := s.questionSource(questionID)
	_, err := s.db.Exec(
		`INSERT INTO doubt_marks(user_id, question_id, source, marked_at)
		 VALUES(?, ?, ?, ?)
		 ON CONFLICT(user_id, question_id) DO UPDATE SET
		 source = excluded.source,
		 marked_at = excluded.marked_at`,
		userID, questionID, source, time.Now().Format(time.RFC3339),
	)
	return err
}

func (s *Store) DoubtIDs(userID string, questionIDs []string) []string {
	if len(questionIDs) == 0 {
		return nil
	}
	allowed := map[string]bool{}
	for _, id := range questionIDs {
		allowed[id] = true
	}
	rows, err := s.db.Query(`SELECT question_id FROM doubt_marks WHERE user_id = ?`, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil && allowed[id] {
			ids = append(ids, id)
		}
	}
	return ids
}

func (s *Store) DoubtBook(userID string) ([]model.WrongQuestion, error) {
	rows, err := s.db.Query(
		`SELECT question_id, source, marked_at FROM doubt_marks WHERE user_id = ? ORDER BY marked_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	var records []model.DoubtRecord
	for rows.Next() {
		var record model.DoubtRecord
		if err := rows.Scan(&record.QuestionID, &record.Source, &record.MarkedAt); err != nil {
			rows.Close()
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	var items []model.WrongQuestion
	for _, record := range records {
		q, ok := s.questionByID(record.QuestionID)
		if !ok {
			continue
		}
		q.Raw = ""
		rec := record
		items = append(items, model.WrongQuestion{Question: q, Doubt: &rec})
	}
	return items, nil
}
