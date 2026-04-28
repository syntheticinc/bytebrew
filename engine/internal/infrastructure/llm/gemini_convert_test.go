package llm

import (
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaMessagesToGemini_UserMessage(t *testing.T) {
	msgs := []*schema.Message{
		{Role: schema.User, Content: "Hello, world!"},
	}

	contents, sysInstr := schemaMessagesToGemini(msgs)

	require.Len(t, contents, 1)
	assert.Nil(t, sysInstr)
	assert.Equal(t, "user", contents[0].Role)
	require.Len(t, contents[0].Parts, 1)
	assert.Equal(t, "Hello, world!", contents[0].Parts[0].Text)
}

func TestSchemaMessagesToGemini_AssistantMessage(t *testing.T) {
	msgs := []*schema.Message{
		{Role: schema.Assistant, Content: "Hi there!"},
	}

	contents, sysInstr := schemaMessagesToGemini(msgs)

	require.Len(t, contents, 1)
	assert.Nil(t, sysInstr)
	assert.Equal(t, "model", contents[0].Role)
	require.Len(t, contents[0].Parts, 1)
	assert.Equal(t, "Hi there!", contents[0].Parts[0].Text)
}

func TestSchemaMessagesToGemini_AssistantWithToolCall(t *testing.T) {
	msgs := []*schema.Message{
		{
			Role:    schema.Assistant,
			Content: "Let me check the weather.",
			ToolCalls: []schema.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: schema.FunctionCall{
						Name:      "get_weather",
						Arguments: `{"location":"NYC"}`,
					},
				},
			},
		},
	}

	contents, _ := schemaMessagesToGemini(msgs)

	require.Len(t, contents, 1)
	assert.Equal(t, "model", contents[0].Role)
	require.Len(t, contents[0].Parts, 2)

	assert.Equal(t, "Let me check the weather.", contents[0].Parts[0].Text)

	fc := contents[0].Parts[1].FunctionCall
	require.NotNil(t, fc)
	assert.Equal(t, "get_weather", fc.Name)
	assert.Equal(t, "NYC", fc.Args["location"])
}

func TestSchemaMessagesToGemini_ToolResult(t *testing.T) {
	msgs := []*schema.Message{
		{
			Role:    schema.Tool,
			Name:    "get_weather",
			Content: `{"temperature": 72}`,
		},
	}

	contents, _ := schemaMessagesToGemini(msgs)

	require.Len(t, contents, 1)
	assert.Equal(t, "user", contents[0].Role)
	require.Len(t, contents[0].Parts, 1)

	fr := contents[0].Parts[0].FunctionResponse
	require.NotNil(t, fr)
	assert.Equal(t, "get_weather", fr.Name)
	assert.Equal(t, `{"temperature": 72}`, fr.Response["content"])
}

func TestSchemaMessagesToGemini_SystemMessage(t *testing.T) {
	msgs := []*schema.Message{
		{Role: schema.System, Content: "You are a helpful assistant."},
		{Role: schema.User, Content: "Hello"},
	}

	contents, sysInstr := schemaMessagesToGemini(msgs)

	require.Len(t, contents, 1)
	require.NotNil(t, sysInstr)
	require.Len(t, sysInstr.Parts, 1)
	assert.Equal(t, "You are a helpful assistant.", sysInstr.Parts[0].Text)

	assert.Equal(t, "user", contents[0].Role)
	assert.Equal(t, "Hello", contents[0].Parts[0].Text)
}

func TestSchemaToolsToGemini(t *testing.T) {
	tools := []*schema.ToolInfo{
		{
			Name: "get_weather",
			Desc: "Get weather for a location",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"location": {
					Type:     "string",
					Desc:     "City name",
					Required: true,
				},
			}),
		},
	}

	geminiTools := schemaToolsToGemini(tools)

	require.Len(t, geminiTools, 1)
	require.Len(t, geminiTools[0].FunctionDeclarations, 1)

	decl := geminiTools[0].FunctionDeclarations[0]
	assert.Equal(t, "get_weather", decl.Name)
	assert.Equal(t, "Get weather for a location", decl.Description)
	assert.NotNil(t, decl.Parameters)
}

func TestSchemaToolsToGemini_NoTools(t *testing.T) {
	result := schemaToolsToGemini(nil)
	assert.Nil(t, result)

	result = schemaToolsToGemini([]*schema.ToolInfo{})
	assert.Nil(t, result)
}

func TestGeminiResponseToSchema_TextResponse(t *testing.T) {
	resp := &geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Role: "model",
					Parts: []geminiPart{
						{Text: "The weather is sunny."},
					},
				},
				FinishReason: "STOP",
			},
		},
	}

	msg := geminiResponseToSchema(resp)

	assert.Equal(t, schema.Assistant, msg.Role)
	assert.Equal(t, "The weather is sunny.", msg.Content)
	assert.Empty(t, msg.ToolCalls)
}

func TestGeminiResponseToSchema_FunctionCallResponse(t *testing.T) {
	resp := &geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Role: "model",
					Parts: []geminiPart{
						{
							FunctionCall: &geminiFunctionCall{
								Name: "get_weather",
								Args: map[string]interface{}{
									"location": "NYC",
								},
							},
						},
					},
				},
			},
		},
	}

	msg := geminiResponseToSchema(resp)

	assert.Equal(t, schema.Assistant, msg.Role)
	assert.Empty(t, msg.Content)
	require.Len(t, msg.ToolCalls, 1)

	tc := msg.ToolCalls[0]
	assert.Equal(t, "get_weather", tc.Function.Name)
	assert.Equal(t, "function", tc.Type)
	assert.JSONEq(t, `{"location":"NYC"}`, tc.Function.Arguments)
}

func TestGeminiResponseToSchema_EmptyCandidates(t *testing.T) {
	resp := &geminiResponse{
		Candidates: []geminiCandidate{},
	}

	msg := geminiResponseToSchema(resp)

	assert.Equal(t, schema.Assistant, msg.Role)
	assert.Empty(t, msg.Content)
}

func TestGeminiResponseToSchema_MixedParts(t *testing.T) {
	resp := &geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Role: "model",
					Parts: []geminiPart{
						{Text: "Let me check."},
						{
							FunctionCall: &geminiFunctionCall{
								Name: "search",
								Args: map[string]interface{}{"query": "test"},
							},
						},
					},
				},
			},
		},
	}

	msg := geminiResponseToSchema(resp)

	assert.Equal(t, "Let me check.", msg.Content)
	require.Len(t, msg.ToolCalls, 1)
	assert.Equal(t, "search", msg.ToolCalls[0].Function.Name)
}

func TestParseToolCallArgs_ValidJSON(t *testing.T) {
	result := parseToolCallArgs(`{"key":"value","num":42}`)

	assert.Equal(t, "value", result["key"])
	assert.Equal(t, float64(42), result["num"])
}

func TestParseToolCallArgs_Empty(t *testing.T) {
	result := parseToolCallArgs("")
	assert.Nil(t, result)
}

func TestParseToolCallArgs_InvalidJSON(t *testing.T) {
	result := parseToolCallArgs("not json")

	assert.Equal(t, "not json", result["raw"])
}

func TestSchemaMessagesToGemini_FullConversation(t *testing.T) {
	msgs := []*schema.Message{
		{Role: schema.System, Content: "You are helpful."},
		{Role: schema.User, Content: "What is the weather?"},
		{
			Role: schema.Assistant,
			ToolCalls: []schema.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: schema.FunctionCall{
						Name:      "get_weather",
						Arguments: `{"location":"London"}`,
					},
				},
			},
		},
		{
			Role:    schema.Tool,
			Name:    "get_weather",
			Content: "Sunny, 20C",
		},
		{Role: schema.Assistant, Content: "The weather in London is sunny, 20C."},
	}

	contents, sysInstr := schemaMessagesToGemini(msgs)

	require.NotNil(t, sysInstr)
	assert.Equal(t, "You are helpful.", sysInstr.Parts[0].Text)

	// System is extracted, so 4 contents: user, assistant(tool call), user(tool result), assistant
	require.Len(t, contents, 4)

	assert.Equal(t, "user", contents[0].Role)
	assert.Equal(t, "model", contents[1].Role)
	assert.Equal(t, "user", contents[2].Role)  // tool result
	assert.Equal(t, "model", contents[3].Role)

	// Tool call in assistant message
	require.NotNil(t, contents[1].Parts[0].FunctionCall)
	assert.Equal(t, "get_weather", contents[1].Parts[0].FunctionCall.Name)

	// Tool result
	require.NotNil(t, contents[2].Parts[0].FunctionResponse)
	assert.Equal(t, "get_weather", contents[2].Parts[0].FunctionResponse.Name)
}
