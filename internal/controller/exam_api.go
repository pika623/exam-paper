package controller

import (
	"exam-paper/internal/model"
	"exam-paper/internal/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	defaultReviewPageSize = 20
	maxReviewPageSize     = 100
)

func (ctl *Controller) Users(c *gin.Context) {
	switch c.Request.Method {
	case http.MethodGet:
		writeJSON(c, gin.H{"users": ctl.store.ListUsers()})
	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
		}
		if !bindJSON(c, &req) {
			return
		}
		user, err := ctl.store.RegisterUser(req.Name)
		if err != nil {
			writeError(c, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(c, gin.H{"user": user})
	}
}

func (ctl *Controller) ClearUser(c *gin.Context) {
	var req struct {
		UserID string `json:"userId"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if err := ctl.store.ClearUser(req.UserID); err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(c, gin.H{"ok": true})
}

func (ctl *Controller) CreateExam(c *gin.Context) {
	var req struct {
		UserID  string   `json:"userId"`
		Sources []string `json:"sources"`
		Count   int      `json:"count"`
	}
	if !bindJSON(c, &req) {
		return
	}
	exam, err := ctl.store.CreateExam(req.UserID, utils.CleanSources(req.Sources), req.Count)
	if err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	ctl.writeExamPayload(c, exam, http.StatusBadRequest)
}

func (ctl *Controller) DeleteExam(c *gin.Context) {
	userID := c.Query("userId")
	examID := c.Query("examId")
	if userID == "" || examID == "" {
		writeError(c, "缺少 userId 或 examId。", http.StatusBadRequest)
		return
	}
	if err := ctl.store.DeleteExam(userID, examID); err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(c, gin.H{"ok": true})
}

func (ctl *Controller) CurrentExam(c *gin.Context) {
	userID := c.Query("userId")
	examID := c.Query("examId")
	var (
		exam model.ExamRecord
		ok   bool
		err  error
	)
	if examID != "" {
		exam, err = ctl.store.ExamByID(userID, examID)
		ok = err == nil
	} else {
		exam, ok, err = ctl.store.CurrentExam(userID)
	}
	if err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	if !ok {
		writeJSON(c, gin.H{"exam": nil})
		return
	}
	ctl.writeExamPayload(c, exam, http.StatusBadRequest)
}

func (ctl *Controller) ExamList(c *gin.Context) {
	exams, err := ctl.store.ListIncompleteExams(c.Query("userId"))
	if err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(c, gin.H{"exams": exams})
}

func (ctl *Controller) ExamQuestions(c *gin.Context) {
	userID := c.Query("userId")
	examID := c.Query("examId")
	if userID == "" || examID == "" {
		writeError(c, "缺少 userId 或 examId。", http.StatusBadRequest)
		return
	}
	center, _ := strconv.Atoi(c.DefaultQuery("center", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "25"))
	exam, err := ctl.store.ExamByID(userID, examID)
	if err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	payload, err := ctl.bank.ExamQuestionsPayload(exam, center, limit)
	if err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(c, payload)
}

func (ctl *Controller) ExamAnswer(c *gin.Context) {
	var req struct {
		UserID     string   `json:"userId"`
		ExamID     string   `json:"examId"`
		QuestionID string   `json:"questionId"`
		Selected   []string `json:"selected"`
		Current    int      `json:"current"`
	}
	if !bindJSON(c, &req) {
		return
	}
	q, ok := ctl.bank.QuestionByID(req.QuestionID)
	if !ok {
		writeError(c, "题目不存在。", http.StatusBadRequest)
		return
	}
	correct := utils.SameChoices(req.Selected, q.Answer)
	exam, answer, err := ctl.store.SaveAnswer(req.UserID, req.ExamID, req.QuestionID, req.Selected, req.Current, correct)
	if err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(c, gin.H{"exam": examState(exam), "answer": answer})
}

func (ctl *Controller) ExamProgress(c *gin.Context) {
	var req struct {
		UserID  string `json:"userId"`
		ExamID  string `json:"examId"`
		Current int    `json:"current"`
	}
	if !bindJSON(c, &req) {
		return
	}
	exam, err := ctl.store.SaveProgress(req.UserID, req.ExamID, req.Current)
	if err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(c, gin.H{"exam": examState(exam)})
}

func examState(exam model.ExamRecord) model.ExamState {
	return model.ExamState{
		ID:        exam.ID,
		Current:   exam.Current,
		Completed: exam.Completed,
		UpdatedAt: exam.UpdatedAt,
	}
}

func (ctl *Controller) ExamWrong(c *gin.Context) {
	offset, limit := reviewPagination(c)
	page, err := ctl.store.ExamWrongQuestionPage(c.Query("userId"), c.Query("examId"), offset, limit)
	if err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(c, page)
}

func (ctl *Controller) WrongPracticeExam(c *gin.Context) {
	var req struct {
		UserID string `json:"userId"`
		Count  int    `json:"count"`
	}
	if !bindJSON(c, &req) {
		return
	}
	exam, err := ctl.store.CreateWrongExam(req.UserID, req.Count)
	if err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	ctl.writeExamPayload(c, exam, http.StatusInternalServerError)
}

func (ctl *Controller) FeaturedWrongPracticeExam(c *gin.Context) {
	var req struct {
		UserID string `json:"userId"`
		Count  int    `json:"count"`
	}
	if !bindJSON(c, &req) {
		return
	}
	exam, err := ctl.store.CreateFeaturedWrongExam(req.UserID, req.Count)
	if err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	ctl.writeExamPayload(c, exam, http.StatusInternalServerError)
}

func (ctl *Controller) DoubtPracticeExam(c *gin.Context) {
	var req struct {
		UserID string `json:"userId"`
		Count  int    `json:"count"`
	}
	if !bindJSON(c, &req) {
		return
	}
	exam, err := ctl.store.CreateDoubtExam(req.UserID, req.Count)
	if err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	ctl.writeExamPayload(c, exam, http.StatusInternalServerError)
}

func (ctl *Controller) QuestionCount(c *gin.Context) {
	var req struct {
		Sources []string `json:"sources"`
	}
	if !bindJSON(c, &req) {
		return
	}
	sources := utils.CleanSources(req.Sources)
	if len(sources) == 0 {
		writeJSON(c, gin.H{"count": 0})
		return
	}
	questions, err := ctl.bank.QuestionsBySources(sources)
	if err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(c, gin.H{"count": len(questions)})
}

func (ctl *Controller) WrongBook(c *gin.Context) {
	offset, limit := reviewPagination(c)
	page, err := ctl.store.WrongBookPage(c.Query("userId"), offset, limit)
	if err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(c, page)
}

func (ctl *Controller) FeaturedWrongBook(c *gin.Context) {
	offset, limit := reviewPagination(c)
	page, err := ctl.store.FeaturedWrongBookPage(c.Query("userId"), offset, limit)
	if err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(c, page)
}

func (ctl *Controller) Doubts(c *gin.Context) {
	items, err := ctl.store.DoubtBook(c.Query("userId"))
	if err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(c, gin.H{"items": items})
}

func (ctl *Controller) DoubtToggle(c *gin.Context) {
	var req struct {
		UserID     string `json:"userId"`
		QuestionID string `json:"questionId"`
		Marked     bool   `json:"marked"`
	}
	if !bindJSON(c, &req) {
		return
	}
	if _, ok := ctl.bank.QuestionByID(req.QuestionID); !ok {
		writeError(c, "题目不存在。", http.StatusBadRequest)
		return
	}
	if err := ctl.store.SetDoubt(req.UserID, req.QuestionID, req.Marked); err != nil {
		writeError(c, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(c, gin.H{"ok": true, "marked": req.Marked})
}

func (ctl *Controller) writeExamPayload(c *gin.Context, exam model.ExamRecord, errorStatus int) {
	payload, err := ctl.bank.ExamPayloadFor(exam)
	if err != nil {
		writeError(c, err.Error(), errorStatus)
		return
	}
	writeJSON(c, payload)
}

func reviewPagination(c *gin.Context) (int, int) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultReviewPageSize)))
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = defaultReviewPageSize
	}
	if limit > maxReviewPageSize {
		limit = maxReviewPageSize
	}
	return offset, limit
}
