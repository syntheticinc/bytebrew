package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// MockChatModel implements model.ToolCallingChatModel
// Returns predefined responses based on scenario and message history
type MockChatModel struct {
	scenario string
}

// NewMockChatModel creates a new MockChatModel
func NewMockChatModel(scenario string) *MockChatModel {
	return &MockChatModel{scenario: scenario}
}

// Generate implements model.ChatModel.Generate
func (m *MockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	hasToolResult := containsToolResult(input)

	switch m.scenario {
	case "echo":
		return textMessage("Hello, world!"), nil

	case "server-tool":
		if hasToolResult {
			// Second call after tool execution
			return textMessage("Task operation complete."), nil
		}
		// First call - request tool
		return toolCallMessage("manage_subtasks", `{"action":"list","task_id":"test-1"}`), nil

	case "reasoning":
		return reasoningMessage("Let me think...", "The answer is 42."), nil

	case "error":
		return nil, fmt.Errorf("mock LLM error: simulated failure")

	case "proxied-read":
		if hasToolResult {
			// Second call after read_file execution
			return textMessage(fmt.Sprintf("File contains: %s", extractLastToolResult(input))), nil
		}
		// First call - request read_file
		return toolCallMessage("read_file", `{"file_path":"src/main.ts"}`), nil

	case "proxied-write":
		if hasToolResult {
			// Second call after write_file execution
			return textMessage("File written successfully."), nil
		}
		// First call - request write_file
		return toolCallMessage("write_file", `{"file_path":"output.txt","content":"hello"}`), nil

	case "proxied-exec":
		if hasToolResult {
			// Second call after execute_command execution
			return textMessage(fmt.Sprintf("Command output: %s", extractLastToolResult(input))), nil
		}
		// First call - request execute_command
		return toolCallMessage("execute_command", `{"command":"echo test"}`), nil

	case "ask-user":
		if hasToolResult {
			// Second call after ask_user execution
			return textMessage(fmt.Sprintf("User said: %s", extractLastToolResult(input))), nil
		}
		// First call - request ask_user with structured questions
		return toolCallMessage("ask_user", `{"questions":"[{\"text\":\"Approve?\",\"options\":[{\"label\":\"approved\"},{\"label\":\"rejected\"}],\"default\":\"approved\"}]"}`), nil

	case "multi-tool":
		toolCount := countToolResults(input)
		switch {
		case toolCount == 0:
			// First call - request read_file for a.ts
			return toolCallMessage("read_file", `{"file_path":"a.ts"}`), nil
		case toolCount == 1:
			// Second call - request read_file for b.ts
			msg := &schema.Message{
				Role: schema.Assistant,
				ToolCalls: []schema.ToolCall{{
					ID:   "call_mock_2",
					Type: "function",
					Function: schema.FunctionCall{
						Name:      "read_file",
						Arguments: `{"file_path":"b.ts"}`,
					},
				}},
				ResponseMeta: &schema.ResponseMeta{
					FinishReason: "tool_calls",
				},
			}
			return msg, nil
		default:
			// Third call (2+ tool results) - final answer
			return textMessage("All files read successfully."), nil
		}

	case "tool-error":
		if hasToolResult {
			// Second call after failed read_file
			return textMessage("Handled error gracefully."), nil
		}
		// First call - request read_file for nonexistent file
		return toolCallMessage("read_file", `{"file_path":"nonexistent.ts"}`), nil

	case "task-create":
		if hasToolResult {
			// Second call after manage_tasks execution
			return textMessage(fmt.Sprintf("Task result: %s", extractLastToolResult(input))), nil
		}
		// First call - request manage_tasks create
		// NOTE: manage_tasks will internally call ask_user via proxy for approval
		return toolCallMessage("manage_tasks", `{"action":"create","title":"Test Task","description":"Implement feature X","acceptance_criteria":["Tests pass","Code reviewed"]}`), nil

	case "proxied-edit":
		if hasToolResult {
			// Second call after edit_file execution
			return textMessage("Edit applied successfully."), nil
		}
		// First call - request edit_file
		return toolCallMessage("edit_file", `{"file_path":"src/app.ts","old_string":"console.log('old')","new_string":"console.log('new')"}`), nil

	case "proxied-tree":
		if hasToolResult {
			// Second call after get_project_tree execution
			return textMessage(fmt.Sprintf("Project structure: %s", extractLastToolResult(input))), nil
		}
		// First call - request get_project_tree
		return toolCallMessage("get_project_tree", `{}`), nil

	case "proxied-search":
		if hasToolResult {
			// Second call after search_code execution
			return textMessage(fmt.Sprintf("Search results: %s", extractLastToolResult(input))), nil
		}
		// First call - request search_code
		return toolCallMessage("search_code", `{"query":"hello world"}`), nil

	case "multi-agent":
		// Differentiate supervisor vs code agent by input content
		if isCodeAgentCall(input) {
			// Code agent receives subtask description as input
			// Simply return final answer
			return textMessage("Code agent: task completed successfully."), nil
		}
		// Supervisor flow
		if hasToolResult {
			return textMessage("All agents completed. Work is done."), nil
		}
		// First call - spawn code agent with pre-created subtask
		return toolCallMessage("spawn_code_agent", `{"action":"spawn","subtask_id":"test-subtask-1"}`), nil

	case "agent-interrupt":
		if isCodeAgentCall(input) {
			// Code agent: slow execution (gives time for user interrupt)
			time.Sleep(5 * time.Second)
			return textMessage("Code agent: task completed."), nil
		}
		// Supervisor flow
		if hasToolResult {
			lastResult := extractLastToolResult(input)
			if strings.Contains(lastResult, "[INTERRUPT]") {
				return textMessage("Supervisor: received user interrupt, handling message."), nil
			}
			return textMessage("Supervisor: all agents completed successfully."), nil
		}
		// First call - spawn code agent (blocking)
		return toolCallMessage("spawn_code_agent", `{"action":"spawn","subtask_id":"test-subtask-1"}`), nil

	case "smart-search":
		return toolCallAndReturn("smart_search", `{"query":"handleError error handling","limit":10}`, "SEARCH_RESULT:", hasToolResult, input)

	case "smart-search-exact":
		return toolCallAndReturn("smart_search", `{"query":"handleError","limit":10}`, "SEARCH_RESULT:", hasToolResult, input)

	case "smart-search-broad":
		return toolCallAndReturn("smart_search", `{"query":"error handling patterns","limit":10}`, "SEARCH_RESULT:", hasToolResult, input)

	case "smart-search-symbol":
		return toolCallAndReturn("smart_search", `{"query":"DomainError","limit":10}`, "SEARCH_RESULT:", hasToolResult, input)

	case "smart-search-cross-file":
		return toolCallAndReturn("smart_search", `{"query":"http handler request","limit":10}`, "SEARCH_RESULT:", hasToolResult, input)

	case "smart-search-no-match":
		return toolCallAndReturn("smart_search", `{"query":"kubernetes deployment yaml","limit":10}`, "SEARCH_RESULT:", hasToolResult, input)

	case "grep-direct":
		return toolCallAndReturn("grep_search", `{"pattern":"func\\s+handle","include":"*.go","limit":10}`, "GREP_RESULT:", hasToolResult, input)

	case "glob-search":
		return toolCallAndReturn("glob", `{"pattern":"**/*.go","limit":10}`, "GLOB_RESULT:", hasToolResult, input)

	case "compare-exact-grep":
		return toolCallAndReturn("grep_search", `{"pattern":"handleError","limit":10}`, "GREP_RESULT:", hasToolResult, input)

	case "compare-broad-grep":
		return toolCallAndReturn("grep_search", `{"pattern":"error|handling","include":"*.go","limit":10}`, "GREP_RESULT:", hasToolResult, input)

	case "compare-symbol-grep":
		return toolCallAndReturn("grep_search", `{"pattern":"DomainError","limit":10}`, "GREP_RESULT:", hasToolResult, input)

	case "compare-cross-grep":
		return toolCallAndReturn("grep_search", `{"pattern":"http.*handler|handler.*request","include":"*.go","ignore_case":true,"limit":10}`, "GREP_RESULT:", hasToolResult, input)

	case "smart-search-empty-query":
		// Tests that smart_search with empty query returns error and model recovers
		toolCount := countToolResults(input)
		if toolCount == 0 {
			// First call: send smart_search with empty query
			return toolCallMessage("smart_search", `{"query":"","limit":10}`), nil
		}
		// Second call: model received error, gives final answer
		return textMessage(fmt.Sprintf("HANDLED_ERROR:%s", extractLastToolResult(input))), nil

	case "grep-no-duplicate":
		// Tests that grep_search tool call appears exactly once (no duplicate from event stream)
		return toolCallAndReturn("grep_search", `{"pattern":"ListenAndServe","limit":10}`, "GREP_RESULT:", hasToolResult, input)

	case "glob-no-duplicate":
		// Tests that glob tool call appears exactly once (no duplicate from event stream)
		return toolCallAndReturn("glob", `{"pattern":"**/*.go","limit":10}`, "GLOB_RESULT:", hasToolResult, input)

	case "lsp-definition":
		// Tests lsp tool pipeline: definition operation round-trip through gRPC proxy
		return toolCallAndReturn("lsp", `{"symbol_name":"TestFunc","operation":"definition"}`, "LSP_RESULT:", hasToolResult, input)

	case "lsp-references":
		return toolCallAndReturn("lsp", `{"symbol_name":"HandleError","operation":"references"}`, "LSP_REFS:", hasToolResult, input)

	case "lsp-implementation":
		return toolCallAndReturn("lsp", `{"symbol_name":"Repository","operation":"implementation"}`, "LSP_IMPL:", hasToolResult, input)

	case "lsp-invalid-op":
		return toolCallAndReturn("lsp", `{"symbol_name":"Foo","operation":"hover"}`, "LSP_ERR:", hasToolResult, input)

	case "lsp-missing-symbol":
		return toolCallAndReturn("lsp", `{"symbol_name":"NonExistentSymbol12345","operation":"definition"}`, "LSP_MISS:", hasToolResult, input)

	case "lsp-multilang":
		// Multi-language LSP test: 4 sequential calls for different languages
		toolCount := countToolResults(input)
		switch {
		case toolCount == 0:
			return toolCallMessageWithID("lsp", `{"symbol_name":"ProcessData","operation":"definition"}`, "call_mock_1"), nil
		case toolCount == 1:
			return toolCallMessageWithID("lsp", `{"symbol_name":"UserService","operation":"definition"}`, "call_mock_2"), nil
		case toolCount == 2:
			return toolCallMessageWithID("lsp", `{"symbol_name":"DataProcessor","operation":"definition"}`, "call_mock_3"), nil
		case toolCount == 3:
			return toolCallMessageWithID("lsp", `{"symbol_name":"Config","operation":"definition"}`, "call_mock_4"), nil
		default:
			results := collectAllToolResults(input)
			return textMessage(fmt.Sprintf("MULTILANG_RESULTS:[%s]", strings.Join(results, "|"))), nil
		}

	case "write-file-go-error":
		// Tests LSP diagnostics: write a Go file with undeclared variable → gopls detects error
		if hasToolResult {
			return textMessage(fmt.Sprintf("WRITE_RESULT:%s", extractLastToolResult(input))), nil
		}
		return toolCallMessage("write_file", `{"file_path":"broken.go","content":"package main\n\nimport \"fmt\"\n\nfunc main() {\n\tx := undefinedVar\n\tfmt.Println(x)\n}\n"}`), nil

	case "lsp-symbol-search":
		// Tests symbolSearch finds symbols across languages after auto-indexing
		toolCount := countToolResults(input)
		switch {
		case toolCount == 0:
			return toolCallMessageWithID("lsp", `{"symbol_name":"greet","operation":"definition"}`, "call_mock_1"), nil
		case toolCount == 1:
			return toolCallMessageWithID("lsp", `{"symbol_name":"Calculator","operation":"definition"}`, "call_mock_2"), nil
		default:
			results := collectAllToolResults(input)
			return textMessage(fmt.Sprintf("SYMBOL_SEARCH:[%s]", strings.Join(results, "|"))), nil
		}

	case "agent-failure":
		// Code agent returns error, supervisor handles it
		if isCodeAgentCall(input) {
			// Code agent: return error
			return nil, fmt.Errorf("mock code agent error: file not found")
		}
		// Supervisor flow
		if hasToolResult {
			return textMessage("Agent failed. Handling error."), nil
		}
		return toolCallMessage("spawn_code_agent", `{"action":"spawn","subtask_id":"test-subtask-1"}`), nil

	case "multi-agent-read":
		// Code agent reads file and returns result
		if isCodeAgentCall(input) {
			if hasToolResult {
				return textMessage(fmt.Sprintf("Code agent read file: %s", extractLastToolResult(input))), nil
			}
			return toolCallMessage("read_file", `{"file_path":"src/main.ts"}`), nil
		}
		// Supervisor flow
		if hasToolResult {
			return textMessage("Agent read the file successfully."), nil
		}
		return toolCallMessage("spawn_code_agent", `{"action":"spawn","subtask_id":"test-subtask-1"}`), nil

	case "persistent-shell":
		// Tests that state (cwd) persists between execute_command calls in persistent session
		toolCount := countToolResults(input)
		switch {
		case toolCount == 0:
			// First call: cd /tmp
			return toolCallMessage("execute_command", `{"command":"cd /tmp"}`), nil
		case toolCount == 1:
			// Second call: pwd (should return /tmp if persistence works)
			return toolCallMessageWithID("execute_command", `{"command":"pwd"}`, "call_mock_2"), nil
		default:
			// Third call: collect all results
			results := collectAllToolResults(input)
			return textMessage(fmt.Sprintf("PERSISTENT_SHELL_RESULTS:[%s]", strings.Join(results, "|"))), nil
		}

	case "background-process":
		// Tests background process: start, list, kill
		toolCount := countToolResults(input)
		switch {
		case toolCount == 0:
			// First call: start background process
			return toolCallMessage("execute_command", `{"command":"tail -f /dev/null","background":true}`), nil
		case toolCount == 1:
			// Second call: list background processes
			return toolCallMessageWithID("execute_command", `{"bg_action":"list"}`, "call_mock_2"), nil
		case toolCount == 2:
			// Third call: kill background process
			return toolCallMessageWithID("execute_command", `{"bg_action":"kill","bg_id":"bg-1"}`, "call_mock_3"), nil
		default:
			// Fourth call: collect all results
			results := collectAllToolResults(input)
			return textMessage(fmt.Sprintf("BACKGROUND_RESULTS:[%s]", strings.Join(results, "|"))), nil
		}

	case "cancel-during-stream":
		return textMessage("This is a response that will be cancelled."), nil

	case "parallel-exec":
		// Tests parallel execute_command: two commands in one response
		toolCount := countToolResults(input)
		if toolCount >= 2 {
			// Both commands completed — return final answer with both results
			results := collectAllToolResults(input)
			return textMessage(fmt.Sprintf("PARALLEL_RESULTS:[%s]", strings.Join(results, "|"))), nil
		}
		// First call — return TWO execute_command calls in ONE response (parallel)
		msg := &schema.Message{
			Role: schema.Assistant,
			ToolCalls: []schema.ToolCall{
				{
					ID:   "call_par_1",
					Type: "function",
					Function: schema.FunctionCall{
						Name:      "execute_command",
						Arguments: `{"command":"echo parallel_a"}`,
					},
				},
				{
					ID:   "call_par_2",
					Type: "function",
					Function: schema.FunctionCall{
						Name:      "execute_command",
						Arguments: `{"command":"echo parallel_b"}`,
					},
				},
			},
			ResponseMeta: &schema.ResponseMeta{
				FinishReason: "tool_calls",
			},
		}
		return msg, nil

	default:
		return textMessage("Unknown scenario"), nil
	}
}

// Stream implements model.ChatModel.Stream
func (m *MockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	msg, err := m.Generate(ctx, input, opts...)
	if err != nil {
		return nil, err
	}

	// Wrap single message in StreamReader
	sr, sw := schema.Pipe[*schema.Message](1)
	go func() {
		defer sw.Close()

		if m.scenario == "cancel-during-stream" {
			select {
			case <-time.After(3 * time.Second):
			case <-ctx.Done():
				return
			}
		}

		sw.Send(msg, nil)
	}()

	return sr, nil
}

// WithTools implements model.ToolCallingChatModel.WithTools
func (m *MockChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	// Mock doesn't bind tools - just returns itself
	return m, nil
}

// BindTools implements model.ChatModel.BindTools (deprecated)
func (m *MockChatModel) BindTools(tools []*schema.ToolInfo) error {
	return nil
}

// Helper functions

func textMessage(content string) *schema.Message {
	return &schema.Message{
		Role:    schema.Assistant,
		Content: content,
		ResponseMeta: &schema.ResponseMeta{
			FinishReason: "stop",
		},
	}
}

func toolCallMessage(toolName, args string) *schema.Message {
	return &schema.Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{{
			ID:   "call_mock_1",
			Type: "function",
			Function: schema.FunctionCall{
				Name:      toolName,
				Arguments: args,
			},
		}},
		ResponseMeta: &schema.ResponseMeta{
			FinishReason: "tool_calls",
		},
	}
}

func reasoningMessage(thinking, answer string) *schema.Message {
	return &schema.Message{
		Role:             schema.Assistant,
		Content:          answer,
		ReasoningContent: thinking,
		ResponseMeta: &schema.ResponseMeta{
			FinishReason: "stop",
		},
	}
}

func containsToolResult(input []*schema.Message) bool {
	for _, msg := range input {
		if msg.Role == schema.Tool {
			return true
		}
	}
	return false
}

func extractLastToolResult(input []*schema.Message) string {
	for i := len(input) - 1; i >= 0; i-- {
		if input[i].Role == schema.Tool {
			return input[i].Content
		}
	}
	return ""
}

func countToolResults(input []*schema.Message) int {
	count := 0
	for _, msg := range input {
		if msg.Role == schema.Tool {
			count++
		}
	}
	return count
}

func isCodeAgentCall(input []*schema.Message) bool {
	for _, msg := range input {
		if msg.Role == schema.User && (strings.HasPrefix(msg.Content, "Subtask:") || strings.Contains(msg.Content, "Subtask:")) {
			return true
		}
	}
	return false
}

// toolCallAndReturn is a helper for simple two-step scenarios:
// step 1: return tool call with given args
// step 2: return text with resultPrefix + last tool result
func toolCallAndReturn(toolName, args, resultPrefix string, hasToolResult bool, input []*schema.Message) (*schema.Message, error) {
	if hasToolResult {
		return textMessage(fmt.Sprintf("%s%s", resultPrefix, extractLastToolResult(input))), nil
	}
	return toolCallMessage(toolName, args), nil
}

// toolCallMessageWithID creates a tool call message with a specific call ID.
// Use this for multi-step scenarios where each call needs a unique ID.
func toolCallMessageWithID(toolName, args, callID string) *schema.Message {
	return &schema.Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{{
			ID:   callID,
			Type: "function",
			Function: schema.FunctionCall{
				Name:      toolName,
				Arguments: args,
			},
		}},
		ResponseMeta: &schema.ResponseMeta{
			FinishReason: "tool_calls",
		},
	}
}

// collectAllToolResults returns the Content of every Tool-role message in input.
func collectAllToolResults(input []*schema.Message) []string {
	var results []string
	for _, msg := range input {
		if msg.Role == schema.Tool {
			results = append(results, msg.Content)
		}
	}
	return results
}
