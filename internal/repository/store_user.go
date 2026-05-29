package repository

import (
	"errors"
	"exam-paper/internal/model"
	"exam-paper/internal/utils"
	"strings"
	"time"
)

func (s *Store) ListUsers() []model.PublicUser {
	rows, err := s.db.Query(`SELECT id, name, created_at FROM users ORDER BY created_at`)
	if err != nil {
		return nil
	}

	var users []model.PublicUser
	for rows.Next() {
		var user model.PublicUser
		if err := rows.Scan(&user.ID, &user.Name, &user.CreatedAt); err != nil {
			continue
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil
	}
	rows.Close()
	for i := range users {
		users[i].CurrentExamID, users[i].CurrentExam = s.currentExamBrief(users[i].ID)
		users[i].AnsweredTotal, users[i].CorrectTotal, users[i].WrongTotal = s.answerTotals(users[i].ID)
		users[i].WrongBookCount = s.wrongBookCount(users[i].ID)
		users[i].FeaturedWrongCount = s.featuredWrongCount(users[i].ID)
		users[i].DoubtCount = s.doubtCount(users[i].ID)
	}
	return users
}

func (s *Store) RegisterUser(name string) (model.PublicUser, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.PublicUser{}, errors.New("请输入账号名称。")
	}
	var existing int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM users WHERE lower(name) = lower(?)`, name).Scan(&existing); err != nil {
		return model.PublicUser{}, err
	}
	if existing > 0 {
		return model.PublicUser{}, errors.New("账号名称已存在。")
	}
	now := time.Now().Format(time.RFC3339)
	user := model.PublicUser{ID: utils.NewID("u"), Name: name, CreatedAt: now}
	_, err := s.db.Exec(`INSERT INTO users(id, name, created_at) VALUES(?, ?, ?)`, user.ID, user.Name, user.CreatedAt)
	if err != nil {
		return model.PublicUser{}, errors.New("账号名称已存在。")
	}
	return user, nil
}

func (s *Store) ClearUser(userID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM answers WHERE exam_id IN (SELECT id FROM exams WHERE user_id = ?)`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM exams WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM wrong_book WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM doubt_marks WHERE user_id = ?`, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) userByID(userID string) (model.UserRecord, error) {
	var user model.UserRecord
	err := s.db.QueryRow(`SELECT id, name, created_at FROM users WHERE id = ?`, userID).Scan(&user.ID, &user.Name, &user.CreatedAt)
	if err != nil {
		return model.UserRecord{}, errors.New("账号不存在。")
	}
	return user, nil
}

func (s *Store) currentExamBrief(userID string) (string, string) {
	var id, title string
	err := s.db.QueryRow(`SELECT id, title FROM exams WHERE user_id = ? AND completed = 0 ORDER BY updated_at DESC LIMIT 1`, userID).Scan(&id, &title)
	if err != nil {
		return "", ""
	}
	return id, title
}

func (s *Store) wrongBookCount(userID string) int {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM wrong_book WHERE user_id = ?`, userID).Scan(&count)
	return count
}

func (s *Store) answerTotals(userID string) (int, int, int) {
	var answered, correct, wrong int
	_ = s.db.QueryRow(
		`SELECT COUNT(*),
		 COALESCE(SUM(CASE WHEN answers.correct = 1 THEN 1 ELSE 0 END), 0),
		 COALESCE(SUM(CASE WHEN answers.correct = 0 THEN 1 ELSE 0 END), 0)
		 FROM answers
		 JOIN exams ON exams.id = answers.exam_id
		 WHERE exams.user_id = ? AND answers.judged = 1`, userID,
	).Scan(&answered, &correct, &wrong)
	return answered, correct, wrong
}

func (s *Store) featuredWrongCount(userID string) int {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM wrong_book WHERE user_id = ? AND wrong_count - correct_count >= -3`, userID).Scan(&count)
	return count
}

func (s *Store) doubtCount(userID string) int {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM doubt_marks WHERE user_id = ?`, userID).Scan(&count)
	return count
}



