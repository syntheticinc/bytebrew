package agents

import (
	"context"
	"testing"
)

func TestNewEnhancedStreamToolCallChecker_NilStream(t *testing.T) {
	ctx := context.Background()
	checker := NewEnhancedStreamToolCallChecker()

	// Test with nil stream
	hasToolCall, err := checker(ctx, nil)
	if err != nil {
		t.Errorf("NewEnhancedStreamToolCallChecker() with nil stream error = %v", err)
		return
	}

	if hasToolCall {
		t.Error("NewEnhancedStreamToolCallChecker() with nil stream returned true, want false")
	}
}

func TestNewEnhancedStreamToolCallChecker_Creation(t *testing.T) {
	// Test that checker can be created
	checker := NewEnhancedStreamToolCallChecker()
	if checker == nil {
		t.Error("NewEnhancedStreamToolCallChecker() returned nil")
	}
}
