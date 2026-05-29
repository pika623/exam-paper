package tests

import (
	"exam-paper/internal/model"
	"exam-paper/internal/repository"
	parser "exam-paper/internal/service/parser"
	"exam-paper/internal/service/questionbank"
	"path/filepath"
	"testing"
)

func TestParseQuestions(t *testing.T) {
	text := `1、Question one.【Single】
A、Option A
B、Option B
C、Option C
D、Option D
答案：B
解析：Explanation one.

2、Question two.【Multi】
A、Alpha
B、Beta
C、Gamma
D、Delta
答案：A C
解析：Explanation two.`

	questions := parser.ParseQuestions(parser.CleanText(text), "sample")
	if len(questions) != 2 {
		t.Fatalf("got %d questions, want 2", len(questions))
	}
	if questions[0].Stem == "" || len(questions[0].Options) != 4 {
		t.Fatalf("question parsed incorrectly: %#v", questions[0])
	}
	if got := questions[0].Answer; len(got) != 1 || got[0] != "B" {
		t.Fatalf("answer = %#v, want B", got)
	}
	if questions[0].Explanation != "Explanation one." {
		t.Fatalf("explanation = %q", questions[0].Explanation)
	}
	if got := questions[1].Answer; len(got) != 2 || got[0] != "A" || got[1] != "C" {
		t.Fatalf("multi answer = %#v, want A C", got)
	}
}

func TestStoreExamWrongBookAndClear(t *testing.T) {
	tmp := t.TempDir()
	store, err := repository.NewStore(filepath.Join(tmp, "exam-paper.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	bank := questionbank.New(tmp, store)
	store.SetQuestionFinder(bank)
	bank.RememberQuestions("mock.docx", []model.Question{
		{ID: "q1", Source: "mock.docx", Stem: "1", Options: []model.Option{{Label: "A", Text: "a"}, {Label: "B", Text: "b"}}, Answer: []string{"A"}},
		{ID: "q2", Source: "mock.docx", Stem: "2", Options: []model.Option{{Label: "A", Text: "a"}, {Label: "B", Text: "b"}}, Answer: []string{"B"}},
	})

	user, err := store.RegisterUser("alice")
	if err != nil {
		t.Fatal(err)
	}
	exam, err := store.CreateExam(user.ID, []string{"mock.docx"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(exam.QuestionIDs) != 2 {
		t.Fatalf("question count = %d, want 2", len(exam.QuestionIDs))
	}
	if exam.Title == "" || exam.Title[:5] != "alice" {
		t.Fatalf("exam title = %q, want alice timestamp prefix", exam.Title)
	}
	_, _, err = store.SaveAnswer(user.ID, exam.ID, "q1", []string{"B"}, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = store.SaveAnswer(user.ID, exam.ID, "q1", []string{"B"}, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	book, err := store.WrongBook(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(book) != 1 || book[0].Wrong.WrongCount != 2 {
		t.Fatalf("wrong book = %#v, want q1 twice", book)
	}
	exams, err := store.ListIncompleteExams(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(exams) != 1 || exams[0].Title != exam.Title {
		t.Fatalf("incomplete exams = %#v", exams)
	}
	if err := store.ClearUser(user.ID); err != nil {
		t.Fatal(err)
	}
	book, err = store.WrongBook(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(book) != 0 {
		t.Fatalf("wrong book after clear = %#v, want empty", book)
	}
}
