package agents

import (
	"context"
	"io"

	"github.com/cloudwego/eino/schema"
)

// StreamToolCallChecker is a function to determine whether the model's streaming output contains tool calls.
type StreamToolCallChecker func(ctx context.Context, modelOutput *schema.StreamReader[*schema.Message]) (bool, error)

// NewEnhancedStreamToolCallChecker creates an enhanced stream tool call checker
// that checks all chunks for tool calls (supports Claude and other models)
func NewEnhancedStreamToolCallChecker() StreamToolCallChecker {
	return func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
		if sr == nil {
			return false, nil
		}
		defer sr.Close()

		// Read all chunks and check for tool calls
		for {
			msg, err := sr.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				return false, err
			}

			// Check if message contains tool calls
			if msg != nil && len(msg.ToolCalls) > 0 {
				return true, nil
			}
		}

		return false, nil
	}
}
