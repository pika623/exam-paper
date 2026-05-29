package model

type Option struct {
	Label string `json:"label"`
	Text  string `json:"text"`
}

type Question struct {
	ID          string   `json:"id"`
	Number      string   `json:"number"`
	Type        string   `json:"type"`
	Source      string   `json:"source"`
	Stem        string   `json:"stem"`
	Options     []Option `json:"options"`
	Answer      []string `json:"answer"`
	AnswerText  string   `json:"answerText"`
	Explanation string   `json:"explanation"`
	Raw         string   `json:"raw,omitempty"`
}

type ParseResult struct {
	Source        string     `json:"source"`
	Path          string     `json:"path"`
	QuestionCount int        `json:"questionCount"`
	Questions     []Question `json:"questions"`
	Preview       string     `json:"preview"`
}

type ParseSummary struct {
	Source        string `json:"source"`
	Path          string `json:"path"`
	QuestionCount int    `json:"questionCount"`
	Preview       string `json:"preview"`
}

type LibraryFile struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Suffix string `json:"suffix"`
	Size   int64  `json:"size"`
}

type UserRecord struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

type ExamRecord struct {
	ID          string                  `json:"id"`
	UserID      string                  `json:"userId"`
	Title       string                  `json:"title"`
	Sources     []string                `json:"sources"`
	QuestionIDs []string                `json:"questionIds"`
	Answers     map[string]AnswerRecord `json:"answers"`
	Current     int                     `json:"current"`
	Completed   bool                    `json:"completed"`
	CreatedAt   string                  `json:"createdAt"`
	UpdatedAt   string                  `json:"updatedAt"`
}

type ExamState struct {
	ID        string `json:"id"`
	Current   int    `json:"current"`
	Completed bool   `json:"completed"`
	UpdatedAt string `json:"updatedAt"`
}

type ExamMeta struct {
	ID        string   `json:"id"`
	UserID    string   `json:"userId"`
	Title     string   `json:"title"`
	Sources   []string `json:"sources"`
	Current   int      `json:"current"`
	Completed bool     `json:"completed"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

type AnswerRecord struct {
	Selected   []string `json:"selected"`
	Correct    bool     `json:"correct"`
	Judged     bool     `json:"judged"`
	AnsweredAt string   `json:"answeredAt"`
}

type WrongRecord struct {
	QuestionID   string   `json:"questionId"`
	Source       string   `json:"source"`
	WrongCount   int      `json:"wrongCount"`
	CorrectCount int      `json:"correctCount"`
	LastWrongAt  string   `json:"lastWrongAt"`
	LastSelected []string `json:"lastSelected"`
}

type DoubtRecord struct {
	QuestionID string `json:"questionId"`
	Source     string `json:"source"`
	MarkedAt   string `json:"markedAt"`
}

type PublicUser struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	CreatedAt          string `json:"createdAt"`
	CurrentExamID      string `json:"currentExamId"`
	CurrentExam        string `json:"currentExam"`
	AnsweredTotal      int    `json:"answeredTotal"`
	CorrectTotal       int    `json:"correctTotal"`
	WrongTotal         int    `json:"wrongTotal"`
	WrongBookCount     int    `json:"wrongBookCount"`
	FeaturedWrongCount int    `json:"featuredWrongCount"`
	DoubtCount         int    `json:"doubtCount"`
}

type ExamSummary struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Current   int    `json:"current"`
	Total     int    `json:"total"`
	Answered  int    `json:"answered"`
	Wrong     int    `json:"wrong"`
	Completed bool   `json:"completed"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type ExamPayload struct {
	Exam            ExamMeta                `json:"exam"`
	Questions       []Question              `json:"questions"`
	Offset          int                     `json:"offset"`
	Total           int                     `json:"total"`
	Answers         map[string]AnswerRecord `json:"answers"`
	Answered        int                     `json:"answered"`
	Correct         int                     `json:"correct"`
	Wrong           int                     `json:"wrong"`
	AnsweredIndexes []int                   `json:"answeredIndexes"`
	WrongIndexes    []int                   `json:"wrongIndexes"`
	Doubts          []string                `json:"doubts"`
}

type ExamQuestionsPayload struct {
	Questions []Question              `json:"questions"`
	Answers   map[string]AnswerRecord `json:"answers"`
	Offset    int                     `json:"offset"`
	Total     int                     `json:"total"`
}

type WrongQuestion struct {
	Question Question     `json:"question"`
	Answer   AnswerRecord `json:"answer"`
	Wrong    WrongRecord  `json:"wrong"`
	Doubt    *DoubtRecord `json:"doubt,omitempty"`
}

type WrongQuestionPage struct {
	Items   []WrongQuestion `json:"items"`
	Total   int             `json:"total"`
	Offset  int             `json:"offset"`
	Limit   int             `json:"limit"`
	HasMore bool            `json:"hasMore"`
}
