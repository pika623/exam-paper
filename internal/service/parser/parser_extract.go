package parser

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"sort"
	"strings"

	"rsc.io/pdf"
)

func extractPDF(path string) (string, error) {
	reader, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("PDF 解析失败：%w", err)
	}
	var pageTexts []string
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		lines := pdfTextLines(page.Content().Text)
		if len(lines) > 0 {
			pageTexts = append(pageTexts, fmt.Sprintf("\n\n--- 第 %d 页 ---\n%s", i, strings.Join(lines, "\n")))
		}
	}
	if len(pageTexts) == 0 {
		return "", errors.New("没有从 PDF 中提取到文字。若这是扫描版 PDF，需要先 OCR。")
	}
	return strings.Join(pageTexts, "\n"), nil
}

func pdfTextLines(items []pdf.Text) []string {
	sort.SliceStable(items, func(i, j int) bool {
		if abs(items[i].Y-items[j].Y) > 2 {
			return items[i].Y > items[j].Y
		}
		return items[i].X < items[j].X
	})

	var (
		lines   []string
		current []pdf.Text
		lastY   float64
	)
	flush := func() {
		if len(current) == 0 {
			return
		}
		sort.SliceStable(current, func(i, j int) bool { return current[i].X < current[j].X })
		var b strings.Builder
		var prev *pdf.Text
		for i := range current {
			item := current[i]
			if prev != nil && item.X-prev.X-prev.W > item.FontSize*0.32 {
				b.WriteByte(' ')
			}
			b.WriteString(item.S)
			prev = &item
		}
		if line := strings.TrimSpace(b.String()); line != "" {
			lines = append(lines, line)
		}
		current = nil
	}

	for _, item := range items {
		if strings.TrimSpace(item.S) == "" {
			continue
		}
		if len(current) == 0 {
			lastY = item.Y
			current = append(current, item)
			continue
		}
		if abs(item.Y-lastY) <= 2 {
			current = append(current, item)
			continue
		}
		flush()
		lastY = item.Y
		current = append(current, item)
	}
	flush()
	return lines
}

func extractDOCX(path string) (string, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("DOCX 解析失败：%w", err)
	}
	defer reader.Close()

	var chunks []string
	for _, file := range reader.File {
		if file.Name != "word/document.xml" && !strings.HasPrefix(file.Name, "word/header") && !strings.HasPrefix(file.Name, "word/footer") {
			continue
		}
		text, err := readDocxXML(file)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(text) != "" {
			chunks = append(chunks, text)
		}
	}
	if len(chunks) == 0 {
		return "", errors.New("没有从 DOCX 中提取到文字。")
	}
	return strings.Join(chunks, "\n\n"), nil
}

func readDocxXML(file *zip.File) (string, error) {
	rc, err := file.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var b strings.Builder
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		switch value := token.(type) {
		case xml.StartElement:
			switch value.Name.Local {
			case "p", "tr", "br", "cr":
				b.WriteByte('\n')
			case "tab":
				b.WriteByte(' ')
			}
		case xml.CharData:
			b.WriteString(html.UnescapeString(string(value)))
		}
	}
	return b.String(), nil
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}


