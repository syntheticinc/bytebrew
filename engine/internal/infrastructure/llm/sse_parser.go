package llm

import (
	"context"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// parseSSEStream reads SSE events from r and sends partial schema.Messages
// through the pipe writer. Returns nil on clean [DONE] termination.
func parseSSEStream(r io.Reader, sw *schema.StreamWriter[*schema.Message]) error {
	scanner := bufio.NewScanner(r)
	// Allow up to 1 MiB per line (SSE lines with large tool call arguments).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return nil
		}

		var chunk openAIResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Skip malformed chunks; log for debugging.
			slog.WarnContext(context.Background(), "proxy SSE: skip malformed chunk", "error", err)
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := &chunk.Choices[0].Delta
		msg := oaiMessageToSchema(delta)

		if sw.Send(msg, nil) {
			// Reader closed the pipe.
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("proxy SSE scan: %w", err)
	}

	return nil
}
