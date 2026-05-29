package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"exam-paper/internal/model"
	"exam-paper/internal/utils"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type sourceMeta struct {
	Size          int64
	ModTime       string
	QuestionCount int
}

func (s *Store) QuestionsBySource(source string) ([]model.Question, bool, error) {
	source = utils.CanonicalSource(source)
	rows, err := s.db.Query(
		`SELECT id, number, type, source, stem, options_json, answer_json, answer_text, explanation, raw
		 FROM questions WHERE source = ? ORDER BY ordinal`, source,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var questions []model.Question
	for rows.Next() {
		q, err := scanQuestion(rows)
		if err != nil {
			return nil, false, err
		}
		questions = append(questions, q)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return questions, len(questions) > 0, nil
}

func (s *Store) LibrarySources() ([]model.LibraryFile, error) {
	rows, err := s.db.Query(`SELECT source, size FROM question_sources ORDER BY source`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []model.LibraryFile
	for rows.Next() {
		var file model.LibraryFile
		if err := rows.Scan(&file.Path, &file.Size); err != nil {
			return nil, err
		}
		file.Path = utils.CanonicalSource(file.Path)
		file.Name = sourceBaseName(file.Path)
		file.Suffix = strings.ToLower(filepath.Ext(file.Path))
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

func sourceBaseName(source string) string {
	source = utils.CanonicalSource(source)
	if idx := strings.LastIndex(source, "/"); idx >= 0 {
		return source[idx+1:]
	}
	return source
}

func (s *Store) QuestionByID(id string) (model.Question, bool, error) {
	row := s.db.QueryRow(
		`SELECT id, number, type, source, stem, options_json, answer_json, answer_text, explanation, raw
		 FROM questions WHERE id = ?`, id,
	)
	q, err := scanQuestion(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Question{}, false, nil
	}
	if err != nil {
		return model.Question{}, false, err
	}
	return q, true, nil
}

func (s *Store) sourceMeta(source string) (sourceMeta, bool, error) {
	source = utils.CanonicalSource(source)
	var meta sourceMeta
	err := s.db.QueryRow(
		`SELECT size, mod_time, question_count FROM question_sources WHERE source = ?`, source,
	).Scan(&meta.Size, &meta.ModTime, &meta.QuestionCount)
	if errors.Is(err, sql.ErrNoRows) {
		return sourceMeta{}, false, nil
	}
	if err != nil {
		return sourceMeta{}, false, err
	}
	return meta, true, nil
}

func (s *Store) SourceCurrent(source string, info os.FileInfo) bool {
	meta, ok, err := s.sourceMeta(source)
	if err != nil || !ok {
		return false
	}
	return meta.Size == info.Size() && meta.ModTime == info.ModTime().Format(time.RFC3339Nano)
}

func (s *Store) ReplaceSourceQuestions(source string, info os.FileInfo, questions []model.Question) error {
	source = utils.CanonicalSource(source)
	now := time.Now().Format(time.RFC3339Nano)
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM questions WHERE source = ?`, source); err != nil {
		return err
	}
	stmt, err := tx.Prepare(
		`INSERT INTO questions(
			id, source, number, type, stem, options_json, answer_json, answer_text, explanation, raw, ordinal, updated_at
		) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			source = excluded.source,
			number = excluded.number,
			type = excluded.type,
			stem = excluded.stem,
			options_json = excluded.options_json,
			answer_json = excluded.answer_json,
			answer_text = excluded.answer_text,
			explanation = excluded.explanation,
			raw = excluded.raw,
			ordinal = excluded.ordinal,
			updated_at = excluded.updated_at`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, q := range questions {
		if _, err := stmt.Exec(
			q.ID, source, q.Number, q.Type, q.Stem, utils.MustJSON(q.Options), utils.MustJSON(q.Answer),
			q.AnswerText, q.Explanation, q.Raw, i+1, now,
		); err != nil {
			return err
		}
	}
	_, err = tx.Exec(
		`INSERT INTO question_sources(source, size, mod_time, question_count, synced_at)
		 VALUES(?, ?, ?, ?, ?)
		 ON CONFLICT(source) DO UPDATE SET
		 size = excluded.size,
		 mod_time = excluded.mod_time,
		 question_count = excluded.question_count,
		 synced_at = excluded.synced_at`,
		source, info.Size(), info.ModTime().Format(time.RFC3339Nano), len(questions), now,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

type questionScanner interface {
	Scan(dest ...any) error
}

func scanQuestion(row questionScanner) (model.Question, error) {
	var (
		q           model.Question
		optionsJSON string
		answerJSON  string
	)
	if err := row.Scan(
		&q.ID, &q.Number, &q.Type, &q.Source, &q.Stem, &optionsJSON, &answerJSON,
		&q.AnswerText, &q.Explanation, &q.Raw,
	); err != nil {
		return model.Question{}, err
	}
	_ = json.Unmarshal([]byte(optionsJSON), &q.Options)
	_ = json.Unmarshal([]byte(answerJSON), &q.Answer)
	return q, nil
}



