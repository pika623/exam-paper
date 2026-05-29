package repository

import (
	"database/sql"
	"exam-paper/internal/model"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db        *sql.DB
	questions QuestionFinder
}

type QuestionFinder interface {
	QuestionsBySources(sources []string) ([]model.Question, error)
	QuestionByID(id string) (model.Question, bool)
}

func (s *Store) SetQuestionFinder(questions QuestionFinder) {
	s.questions = questions
}

func NewStore(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)
	s := &Store{db: db}
	if err := s.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() {
	if s != nil && s.db != nil {
		_ = s.db.Close()
	}
}

func (s *Store) init() error {
	statements := []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA journal_mode = WAL`,
		`PRAGMA busy_timeout = 8000`,
		`PRAGMA synchronous = NORMAL`,
		`PRAGMA wal_autocheckpoint = 256`,
		`PRAGMA journal_size_limit = 1048576`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS exams (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			title TEXT NOT NULL,
			sources_json TEXT NOT NULL,
			question_ids_json TEXT NOT NULL,
			current_index INTEGER NOT NULL DEFAULT 0,
			completed INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS answers (
			exam_id TEXT NOT NULL,
			question_id TEXT NOT NULL,
			selected_json TEXT NOT NULL,
			correct INTEGER NOT NULL,
			judged INTEGER NOT NULL,
			answered_at TEXT NOT NULL,
			PRIMARY KEY(exam_id, question_id),
			FOREIGN KEY(exam_id) REFERENCES exams(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS wrong_book (
			user_id TEXT NOT NULL,
			question_id TEXT NOT NULL,
			source TEXT NOT NULL,
			wrong_count INTEGER NOT NULL,
			correct_count INTEGER NOT NULL DEFAULT 0,
			last_wrong_at TEXT NOT NULL,
			last_selected_json TEXT NOT NULL,
			PRIMARY KEY(user_id, question_id),
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS doubt_marks (
			user_id TEXT NOT NULL,
			question_id TEXT NOT NULL,
			source TEXT NOT NULL,
			marked_at TEXT NOT NULL,
			PRIMARY KEY(user_id, question_id),
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS question_sources (
			source TEXT PRIMARY KEY,
			size INTEGER NOT NULL,
			mod_time TEXT NOT NULL,
			question_count INTEGER NOT NULL,
			synced_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS questions (
			id TEXT PRIMARY KEY,
			source TEXT NOT NULL,
			number TEXT NOT NULL,
			type TEXT NOT NULL,
			stem TEXT NOT NULL,
			options_json TEXT NOT NULL,
			answer_json TEXT NOT NULL,
			answer_text TEXT NOT NULL,
			explanation TEXT NOT NULL,
			raw TEXT NOT NULL,
			ordinal INTEGER NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_questions_source_ordinal ON questions(source, ordinal)`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}
	if err := s.ensureColumn("wrong_book", "correct_count", `ALTER TABLE wrong_book ADD COLUMN correct_count INTEGER NOT NULL DEFAULT 0`); err != nil {
		return err
	}
	if err := s.cleanupBackslashQuestionSources(); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureColumn(table string, column string, statement string) error {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.Exec(statement)
	return err
}

func (s *Store) cleanupBackslashQuestionSources() error {
	if _, err := s.db.Exec(`DELETE FROM questions WHERE source LIKE '%\%'`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM question_sources WHERE source LIKE '%\%'`); err != nil {
		return err
	}
	return nil
}

func (s *Store) Checkpoint() error {
	_, err := s.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}



