package api

import (
	"exam-paper/internal/controller"
	"exam-paper/internal/repository"
	"exam-paper/internal/service/questionbank"

	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	RootDir string
	DataDir string
	Store   *repository.Store
	Bank    *questionbank.Service
}

func NewRouter(deps Dependencies) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	ctrl := controller.New(deps.RootDir, deps.DataDir, deps.Store, deps.Bank)

	r.GET("/api/library", ctrl.Library)
	r.GET("/api/parse", ctrl.Parse)
	r.POST("/api/import", ctrl.Import)
	r.GET("/api/users", ctrl.Users)
	r.POST("/api/users", ctrl.Users)
	r.POST("/api/users/clear", ctrl.ClearUser)
	r.POST("/api/exams", ctrl.CreateExam)
	r.DELETE("/api/exams", ctrl.DeleteExam)
	r.GET("/api/exams/current", ctrl.CurrentExam)
	r.GET("/api/exams/questions", ctrl.ExamQuestions)
	r.GET("/api/exams/list", ctrl.ExamList)
	r.POST("/api/exams/answer", ctrl.ExamAnswer)
	r.POST("/api/exams/progress", ctrl.ExamProgress)
	r.GET("/api/exams/wrong", ctrl.ExamWrong)
	r.POST("/api/exams/wrong-practice", ctrl.WrongPracticeExam)
	r.POST("/api/exams/featured-wrong-practice", ctrl.FeaturedWrongPracticeExam)
	r.POST("/api/exams/doubt-practice", ctrl.DoubtPracticeExam)
	r.POST("/api/questions/count", ctrl.QuestionCount)
	r.GET("/api/wrong-book", ctrl.WrongBook)
	r.GET("/api/featured-wrong-book", ctrl.FeaturedWrongBook)
	r.GET("/api/doubts", ctrl.Doubts)
	r.POST("/api/doubts/toggle", ctrl.DoubtToggle)
	r.NoRoute(ctrl.Static)

	return r
}
