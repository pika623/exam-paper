package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"exam-paper/internal/model"
	"exam-paper/internal/utils"
	"fmt"
	"time"
)

func (s *Store) CreateExam(userID string, sources []string, count int) (model.ExamRecord, error) {
	user, err := s.userByID(userID)
	if err != nil {
		return model.ExamRecord{}, err
	}
	if len(sources) == 0 {
		return model.ExamRecord{}, errors.New("请选择至少一个题库来源。")
	}
	if count <= 0 {
		return model.ExamRecord{}, errors.New("题目数量必须大于 0。")
	}
	if s.questions == nil {
		return model.ExamRecord{}, errors.New("题库服务未初始化。")
	}
	all, err := s.questions.QuestionsBySources(sources)
	if err != nil {
		return model.ExamRecord{}, err
	}
	if len(all) == 0 {
		return model.ExamRecord{}, errors.New("所选题库没有解析到题目。")
	}
	if count > len(all) {
		count = len(all)
	}
	utils.PreferWrongThenShuffle(s.WrongCounts(userID), all)
	ids := make([]string, 0, count)
	for _, q := range all[:count] {
		ids = append(ids, q.ID)
	}

	now := time.Now()
	exam := model.ExamRecord{
		ID:          utils.NewID("e"),
		UserID:      userID,
		Title:       fmt.Sprintf("%s_%s", user.Name, now.Format("20060102_150405")),
		Sources:     append([]string(nil), sources...),
		QuestionIDs: ids,
		Answers:     map[string]model.AnswerRecord{},
		CreatedAt:   now.Format(time.RFC3339),
		UpdatedAt:   now.Format(time.RFC3339),
	}
	_, err = s.db.Exec(
		`INSERT INTO exams(id, user_id, title, sources_json, question_ids_json, current_index, completed, created_at, updated_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		exam.ID, exam.UserID, exam.Title, utils.MustJSON(exam.Sources), utils.MustJSON(exam.QuestionIDs),
		exam.Current, utils.BoolInt(exam.Completed), exam.CreatedAt, exam.UpdatedAt,
	)
	return exam, err
}

func (s *Store) CreateWrongExam(userID string, count int) (model.ExamRecord, error) {
	user, err := s.userByID(userID)
	if err != nil {
		return model.ExamRecord{}, err
	}
	rows, err := s.db.Query(
		`SELECT question_id, source FROM wrong_book WHERE user_id = ? ORDER BY wrong_count DESC, last_wrong_at DESC`, userID,
	)
	if err != nil {
		return model.ExamRecord{}, err
	}

	var (
		ids     []string
		sources []string
		seenSrc = map[string]bool{}
	)
	for rows.Next() {
		var questionID, source string
		if err := rows.Scan(&questionID, &source); err != nil {
			rows.Close()
			return model.ExamRecord{}, err
		}
		ids = append(ids, questionID)
		if source != "" && !seenSrc[source] {
			seenSrc[source] = true
			sources = append(sources, source)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return model.ExamRecord{}, err
	}
	rows.Close()
	ids = s.ExistingQuestionIDs(ids, count)
	if len(ids) == 0 {
		return model.ExamRecord{}, errors.New("当前账号还没有可练习的错题。")
	}

	now := time.Now()
	exam := model.ExamRecord{
		ID:          utils.NewID("e"),
		UserID:      userID,
		Title:       fmt.Sprintf("%s_%s_错题集", user.Name, now.Format("20060102_150405")),
		Sources:     sources,
		QuestionIDs: ids,
		Answers:     map[string]model.AnswerRecord{},
		CreatedAt:   now.Format(time.RFC3339),
		UpdatedAt:   now.Format(time.RFC3339),
	}
	_, err = s.db.Exec(
		`INSERT INTO exams(id, user_id, title, sources_json, question_ids_json, current_index, completed, created_at, updated_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		exam.ID, exam.UserID, exam.Title, utils.MustJSON(exam.Sources), utils.MustJSON(exam.QuestionIDs),
		exam.Current, utils.BoolInt(exam.Completed), exam.CreatedAt, exam.UpdatedAt,
	)
	return exam, err
}

func (s *Store) CreateFeaturedWrongExam(userID string, count int) (model.ExamRecord, error) {
	user, err := s.userByID(userID)
	if err != nil {
		return model.ExamRecord{}, err
	}
	rows, err := s.db.Query(
		`SELECT question_id, source FROM wrong_book
		 WHERE user_id = ? AND wrong_count - correct_count >= -3
		 ORDER BY (wrong_count - correct_count) DESC, wrong_count DESC, last_wrong_at DESC`, userID,
	)
	if err != nil {
		return model.ExamRecord{}, err
	}
	ids, sources, err := collectPracticeRows(rows, count)
	if err != nil {
		return model.ExamRecord{}, err
	}
	ids = s.ExistingQuestionIDs(ids, count)
	if len(ids) == 0 {
		return model.ExamRecord{}, errors.New("当前账号还没有可练习的精选错题。")
	}
	return s.insertPracticeExam(user.ID, user.Name, "精选错题集", sources, ids)
}

func (s *Store) CreateDoubtExam(userID string, count int) (model.ExamRecord, error) {
	user, err := s.userByID(userID)
	if err != nil {
		return model.ExamRecord{}, err
	}
	rows, err := s.db.Query(
		`SELECT question_id, source FROM doubt_marks WHERE user_id = ? ORDER BY marked_at DESC`, userID,
	)
	if err != nil {
		return model.ExamRecord{}, err
	}
	ids, sources, err := collectPracticeRows(rows, count)
	if err != nil {
		return model.ExamRecord{}, err
	}
	ids = s.ExistingQuestionIDs(ids, count)
	if len(ids) == 0 {
		return model.ExamRecord{}, errors.New("当前账号还没有可练习的疑问题。")
	}
	return s.insertPracticeExam(user.ID, user.Name, "疑问题", sources, ids)
}

func (s *Store) ListIncompleteExams(userID string) ([]model.ExamSummary, error) {
	rows, err := s.db.Query(
		`SELECT id, title, question_ids_json, current_index, completed, created_at, updated_at
		 FROM exams WHERE user_id = ? AND completed = 0 ORDER BY updated_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}

	var exams []model.ExamSummary
	for rows.Next() {
		var (
			summary   model.ExamSummary
			idsJSON   string
			completed int
		)
		if err := rows.Scan(&summary.ID, &summary.Title, &idsJSON, &summary.Current, &completed, &summary.CreatedAt, &summary.UpdatedAt); err != nil {
			return nil, err
		}
		var ids []string
		_ = json.Unmarshal([]byte(idsJSON), &ids)
		summary.Total = len(ids)
		summary.Completed = completed == 1
		exams = append(exams, summary)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for i := range exams {
		exams[i].Answered = s.answerCount(exams[i].ID)
		exams[i].Wrong = s.examWrongCount(exams[i].ID)
	}
	return exams, nil
}

func (s *Store) CurrentExam(userID string) (model.ExamRecord, bool, error) {
	var id string
	err := s.db.QueryRow(
		`SELECT id FROM exams WHERE user_id = ? AND completed = 0 ORDER BY updated_at DESC LIMIT 1`, userID,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.ExamRecord{}, false, nil
	}
	if err != nil {
		return model.ExamRecord{}, false, err
	}
	exam, err := s.ExamByID(userID, id)
	return exam, err == nil, err
}

func (s *Store) ExamByID(userID string, examID string) (model.ExamRecord, error) {
	var (
		exam        model.ExamRecord
		sourcesJSON string
		idsJSON     string
		completed   int
	)
	err := s.db.QueryRow(
		`SELECT id, user_id, title, sources_json, question_ids_json, current_index, completed, created_at, updated_at
		 FROM exams WHERE user_id = ? AND id = ?`, userID, examID,
	).Scan(&exam.ID, &exam.UserID, &exam.Title, &sourcesJSON, &idsJSON, &exam.Current, &completed, &exam.CreatedAt, &exam.UpdatedAt)
	if err != nil {
		return model.ExamRecord{}, errors.New("试卷不存在。")
	}
	_ = json.Unmarshal([]byte(sourcesJSON), &exam.Sources)
	_ = json.Unmarshal([]byte(idsJSON), &exam.QuestionIDs)
	exam.Completed = completed == 1
	exam.Answers = s.answersForExam(exam.ID)
	return exam, nil
}

func (s *Store) DeleteExam(userID string, examID string) error {
	result, err := s.db.Exec(`DELETE FROM exams WHERE user_id = ? AND id = ?`, userID, examID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("试卷不存在。")
	}
	return nil
}

func (s *Store) SaveAnswer(userID string, examID string, questionID string, selected []string, current int, correct bool) (model.ExamRecord, model.AnswerRecord, error) {
	exam, err := s.ExamByID(userID, examID)
	if err != nil {
		return model.ExamRecord{}, model.AnswerRecord{}, err
	}
	now := time.Now().Format(time.RFC3339)
	answer := model.AnswerRecord{
		Selected:   utils.NormalizeChoiceStrings(selected),
		Correct:    correct,
		Judged:     true,
		AnsweredAt: now,
	}
	source := ""
	if !correct {
		source = s.questionSource(questionID)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return model.ExamRecord{}, model.AnswerRecord{}, err
	}
	defer tx.Rollback()
	_, err = tx.Exec(
		`INSERT INTO answers(exam_id, question_id, selected_json, correct, judged, answered_at)
		 VALUES(?, ?, ?, ?, ?, ?)
		 ON CONFLICT(exam_id, question_id) DO UPDATE SET
		 selected_json = excluded.selected_json, correct = excluded.correct, judged = excluded.judged, answered_at = excluded.answered_at`,
		examID, questionID, utils.MustJSON(answer.Selected), utils.BoolInt(answer.Correct), utils.BoolInt(answer.Judged), answer.AnsweredAt,
	)
	if err != nil {
		return model.ExamRecord{}, model.AnswerRecord{}, err
	}
	completed := s.answerCountTx(tx, examID) >= len(exam.QuestionIDs)
	_, err = tx.Exec(`UPDATE exams SET current_index = ?, completed = ?, updated_at = ? WHERE id = ?`, current, utils.BoolInt(completed), now, examID)
	if err != nil {
		return model.ExamRecord{}, model.AnswerRecord{}, err
	}
	if !correct {
		_, err = tx.Exec(
			`INSERT INTO wrong_book(user_id, question_id, source, wrong_count, correct_count, last_wrong_at, last_selected_json)
			 VALUES(?, ?, ?, 1, 0, ?, ?)
			 ON CONFLICT(user_id, question_id) DO UPDATE SET
			 wrong_count = wrong_count + 1, source = excluded.source, last_wrong_at = excluded.last_wrong_at, last_selected_json = excluded.last_selected_json`,
			userID, questionID, source, now, utils.MustJSON(answer.Selected),
		)
		if err != nil {
			return model.ExamRecord{}, model.AnswerRecord{}, err
		}
	} else {
		_, err = tx.Exec(`UPDATE wrong_book SET correct_count = correct_count + 1 WHERE user_id = ? AND question_id = ?`, userID, questionID)
		if err != nil {
			return model.ExamRecord{}, model.AnswerRecord{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return model.ExamRecord{}, model.AnswerRecord{}, err
	}
	exam, err = s.ExamByID(userID, examID)
	return exam, answer, err
}

func (s *Store) SaveProgress(userID string, examID string, current int) (model.ExamRecord, error) {
	exam, err := s.ExamByID(userID, examID)
	if err != nil {
		return model.ExamRecord{}, err
	}
	if current < 0 {
		current = 0
	}
	if current >= len(exam.QuestionIDs) {
		current = len(exam.QuestionIDs) - 1
	}
	now := time.Now().Format(time.RFC3339)
	_, err = s.db.Exec(`UPDATE exams SET current_index = ?, updated_at = ? WHERE id = ?`, current, now, examID)
	if err != nil {
		return model.ExamRecord{}, err
	}
	return s.ExamByID(userID, examID)
}

func (s *Store) answerCount(examID string) int {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM answers WHERE exam_id = ? AND judged = 1`, examID).Scan(&count)
	return count
}

func (s *Store) examWrongCount(examID string) int {
	var count int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM answers WHERE exam_id = ? AND judged = 1 AND correct = 0`, examID).Scan(&count)
	return count
}

func (s *Store) answerCountTx(tx *sql.Tx, examID string) int {
	var count int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM answers WHERE exam_id = ? AND judged = 1`, examID).Scan(&count)
	return count
}

func (s *Store) answersForExam(examID string) map[string]model.AnswerRecord {
	rows, err := s.db.Query(`SELECT question_id, selected_json, correct, judged, answered_at FROM answers WHERE exam_id = ?`, examID)
	if err != nil {
		return map[string]model.AnswerRecord{}
	}
	defer rows.Close()

	answers := map[string]model.AnswerRecord{}
	for rows.Next() {
		var (
			questionID   string
			selectedJSON string
			correct      int
			judged       int
			answer       model.AnswerRecord
		)
		if err := rows.Scan(&questionID, &selectedJSON, &correct, &judged, &answer.AnsweredAt); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(selectedJSON), &answer.Selected)
		answer.Correct = correct == 1
		answer.Judged = judged == 1
		answers[questionID] = answer
	}
	return answers
}

func collectPracticeRows(rows *sql.Rows, count int) ([]string, []string, error) {
	var (
		ids     []string
		sources []string
		seenSrc = map[string]bool{}
	)
	for rows.Next() {
		var questionID, source string
		if err := rows.Scan(&questionID, &source); err != nil {
			rows.Close()
			return nil, nil, err
		}
		ids = append(ids, questionID)
		if source != "" && !seenSrc[source] {
			seenSrc[source] = true
			sources = append(sources, source)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	rows.Close()
	return ids, sources, nil
}

func (s *Store) insertPracticeExam(userID string, userName string, suffix string, sources []string, ids []string) (model.ExamRecord, error) {
	now := time.Now()
	exam := model.ExamRecord{
		ID:          utils.NewID("e"),
		UserID:      userID,
		Title:       fmt.Sprintf("%s_%s_%s", userName, now.Format("20060102_150405"), suffix),
		Sources:     sources,
		QuestionIDs: ids,
		Answers:     map[string]model.AnswerRecord{},
		CreatedAt:   now.Format(time.RFC3339),
		UpdatedAt:   now.Format(time.RFC3339),
	}
	_, err := s.db.Exec(
		`INSERT INTO exams(id, user_id, title, sources_json, question_ids_json, current_index, completed, created_at, updated_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		exam.ID, exam.UserID, exam.Title, utils.MustJSON(exam.Sources), utils.MustJSON(exam.QuestionIDs),
		exam.Current, utils.BoolInt(exam.Completed), exam.CreatedAt, exam.UpdatedAt,
	)
	return exam, err
}



