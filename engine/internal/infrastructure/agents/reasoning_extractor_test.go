package agents

import (
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

func TestReasoningExtractor_WithReasoningContent(t *testing.T) {
	extractor := NewReasoningExtractor()

	msg := &schema.Message{
		Role:             schema.Assistant,
		Content:          "Let me help you",
		ReasoningContent: "First, I need to analyze the problem step by step...",
	}

	reasoning, found := extractor.ExtractReasoning(msg)
	assert.True(t, found)
	assert.Equal(t, "First, I need to analyze the problem step by step...", reasoning)
}

func TestReasoningExtractor_NoReasoning(t *testing.T) {
	extractor := NewReasoningExtractor()

	msg := &schema.Message{
		Role:    schema.Assistant,
		Content: "Just a simple answer",
	}

	reasoning, found := extractor.ExtractReasoning(msg)
	assert.False(t, found)
	assert.Empty(t, reasoning)
}

func TestReasoningExtractor_EmptyReasoning(t *testing.T) {
	extractor := NewReasoningExtractor()

	msg := &schema.Message{
		Role:             schema.Assistant,
		Content:          "Answer",
		ReasoningContent: "",
	}

	reasoning, found := extractor.ExtractReasoning(msg)
	assert.False(t, found)
	assert.Empty(t, reasoning)
}

func TestReasoningExtractor_NilMessage(t *testing.T) {
	extractor := NewReasoningExtractor()

	reasoning, found := extractor.ExtractReasoning(nil)
	assert.False(t, found)
	assert.Empty(t, reasoning)
}

func TestCleanReasoningContent_Normal(t *testing.T) {
	content := "Normal reasoning content without quotes"
	assert.Equal(t, content, cleanReasoningContent(content))
}

func TestCleanReasoningContent_GarbledQuotes(t *testing.T) {
	// OpenRouter streaming produces garbled content like this:
	// each chunk is a JSON string, concatenated together. Test multiple
	// scripts to ensure UTF-8 byte handling stays correct outside ASCII.
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"cyrillic", `"Пользователь""просит""помощь"`, "Пользовательпроситпомощь"},
		{"chinese", `"用户""请求""帮助"`, "用户请求帮助"},
		{"arabic", `"المستخدم""يطلب""المساعدة"`, "المستخدميطلبالمساعدة"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleaned := cleanReasoningContent(tt.input)
			assert.Equal(t, tt.expected, cleaned)
		})
	}
}

func TestCleanReasoningContent_JSONString(t *testing.T) {
	content := `"A valid JSON string"`
	cleaned := cleanReasoningContent(content)
	assert.Equal(t, "A valid JSON string", cleaned)
}
