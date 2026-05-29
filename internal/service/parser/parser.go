package parser

import (
	"errors"
	"exam-paper/internal/model"
	"exam-paper/internal/utils"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	numberRE   = regexp.MustCompile(`(?m)(?:^|[\n ])\s*(\d{1,4})\s*[、.．]\s*`)
	answerRE   = regexp.MustCompile(`(?i)(参考答案|正确答案|答案)\s*[:：]`)
	explainRE  = regexp.MustCompile(`(?i)(答案解析|解析|详解)\s*[:：]`)
	typeRE     = regexp.MustCompile(`[【\[]\s*([^】\]]*?(单选|多选|判断|选择|题)[^】\]]*?)\s*[】\]]`)
	optionRE   = regexp.MustCompile(`(?m)(^|[\n\s])([A-HＡ-Ｈ])\s*[、.．\)）:：]\s*`)
	answerTok  = regexp.MustCompile(`(?i)[A-HＡ-Ｈ]|正确|错误|对|错|是|否|√|×|True|False`)
	privateUse = regexp.MustCompile(`[\x{e000}-\x{f8ff}]`)
)

func ParseQuestionFile(path string, source string) (model.ParseResult, error) {
	source = utils.CanonicalSource(source)
	ext := strings.ToLower(filepath.Ext(path))
	if !utils.FileTypes[ext] {
		return model.ParseResult{}, errors.New("只支持 PDF 或 DOCX 文件。")
	}
	if _, err := os.Stat(path); err != nil {
		return model.ParseResult{}, errors.New("文件不存在。")
	}

	var (
		text string
		err  error
	)
	switch ext {
	case ".pdf":
		text, err = extractPDF(path)
	case ".docx":
		text, err = extractDOCX(path)
	}
	if err != nil {
		return model.ParseResult{}, err
	}
	text = CleanText(text)
	questions := parseQuestions(text, source)
	return model.ParseResult{
		Source:        source,
		Path:          path,
		QuestionCount: len(questions),
		Questions:     questions,
		Preview:       truncateRunes(text, 1000),
	}, nil
}


