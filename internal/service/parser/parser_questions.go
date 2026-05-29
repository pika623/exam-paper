package parser

import (
	"exam-paper/internal/model"
	"strings"
)

func ParseQuestions(text string, source string) []model.Question {
	return parseQuestions(text, source)
}

func parseQuestions(text string, source string) []model.Question {
	matches := numberRE.FindAllStringSubmatchIndex(text, -1)
	var questions []model.Question
	for i, match := range matches {
		start := match[0]
		end := len(text)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		block := strings.TrimSpace(text[start:end])
		if !answerRE.MatchString(block) || len([]rune(block)) < 20 {
			continue
		}
		number := text[match[2]:match[3]]
		if q, ok := parseQuestionBlock(block, source, len(questions)+1, number); ok {
			questions = append(questions, q)
		}
	}
	return questions
}

func parseQuestionBlock(block string, source string, ordinal int, number string) (model.Question, bool) {
	body := strings.TrimSpace(numberRE.ReplaceAllString(block, ""))
	answerLoc := answerRE.FindStringIndex(body)
	if answerLoc == nil {
		return model.Question{}, false
	}
	questionPart := strings.TrimSpace(body[:answerLoc[0]])
	tail := strings.TrimSpace(body[answerLoc[1]:])
	answerText := tail
	explanation := ""
	if loc := explainRE.FindStringIndex(tail); loc != nil {
		answerText = strings.TrimSpace(tail[:loc[0]])
		explanation = strings.TrimSpace(tail[loc[1]:])
	}

	qType := ""
	if loc := typeRE.FindStringSubmatchIndex(questionPart); loc != nil {
		qType = strings.TrimSpace(questionPart[loc[2]:loc[3]])
		questionPart = strings.TrimSpace(questionPart[:loc[0]] + questionPart[loc[1]:])
	}
	stem, options := parseOptions(questionPart)
	answer := normalizeAnswer(answerText)
	if stem == "" || answerText == "" {
		return model.Question{}, false
	}
	explanation = explainRE.ReplaceAllString(explanation, "")
	if qType == "" {
		qType = guessType(answer, options)
	}
	return model.Question{
		ID:          questionID(source, ordinal, stem, answerText),
		Number:      number,
		Type:        qType,
		Source:      source,
		Stem:        strings.TrimSpace(stem),
		Options:     options,
		Answer:      answer,
		AnswerText:  strings.TrimSpace(answerText),
		Explanation: strings.TrimSpace(explanation),
		Raw:         block,
	}, true
}

func parseOptions(text string) (string, []model.Option) {
	matches := optionRE.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return strings.TrimSpace(text), nil
	}
	stem := strings.TrimSpace(text[:matches[0][0]])
	options := make([]model.Option, 0, len(matches))
	for i, match := range matches {
		start := match[1]
		if start < 0 {
			start = match[0]
		}
		next := len(text)
		if i+1 < len(matches) {
			next = matches[i+1][0]
		}
		label := normalizeLabel(text[match[4]:match[5]])
		value := strings.TrimSpace(text[start:next])
		value = optionRE.ReplaceAllString(value, "")
		value = collapseInline(value)
		if value != "" {
			options = append(options, model.Option{Label: label, Text: value})
		}
	}
	return stem, options
}

func normalizeAnswer(text string) []string {
	matches := answerTok.FindAllString(text, -1)
	var result []string
	seen := map[string]bool{}
	for _, match := range matches {
		token := normalizeToken(match)
		if token == "" || seen[token] {
			continue
		}
		seen[token] = true
		result = append(result, token)
	}
	return result
}

func normalizeToken(token string) string {
	token = strings.ToUpper(strings.TrimSpace(token))
	token = strings.Map(func(r rune) rune {
		if r >= 'Ａ' && r <= 'Ｈ' {
			return r - 'Ａ' + 'A'
		}
		return r
	}, token)
	switch token {
	case "正确", "对", "是", "√", "TRUE":
		return "TRUE"
	case "错误", "错", "否", "×", "FALSE":
		return "FALSE"
	}
	if len(token) == 1 && token[0] >= 'A' && token[0] <= 'H' {
		return token
	}
	return ""
}

func normalizeLabel(label string) string {
	return normalizeToken(label)
}

func guessType(answer []string, options []model.Option) string {
	if len(answer) == 1 && (answer[0] == "TRUE" || answer[0] == "FALSE") {
		return "判断题"
	}
	if len(answer) > 1 {
		return "多选题"
	}
	if len(options) > 0 {
		return "单选题"
	}
	return "简答题"
}


