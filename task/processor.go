package task

import (
	"strings"
	"unicode/utf8"
)

type TaskProcessor interface {
	Validate(input TaskInput) error
	Process(input TaskInput) (TaskResult, error)
}

type TextAnalyzer struct{}

func NewTextAnalyzer() *TextAnalyzer {
	return &TextAnalyzer{}
}

func (a *TextAnalyzer) Validate(input TaskInput) error {
	if strings.TrimSpace(input.Text) == "" {
		return ErrInvalidInput
	}
	return nil
}

func (a *TextAnalyzer) Process(input TaskInput) (TaskResult, error) {
	if err := a.Validate(input); err != nil {
		return TaskResult{}, err
	}

	text := input.Text
	words := strings.Fields(text)
	wordCount := len(words)
	charCount := len(text)
	runeCount := utf8.RuneCountInString(text)

	lineCount := 0
	if len(text) > 0 {
		lineCount = strings.Count(text, "\n") + 1
	}

	return TaskResult{
		WordCount:      wordCount,
		CharacterCount: charCount,
		RuneCount:      runeCount,
		LineCount:      lineCount,
	}, nil
}
