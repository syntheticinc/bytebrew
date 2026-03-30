# Response: Tool Result Truncation Bug

**Date:** 2026-03-30
**In response to:** `bytebrew-tool-result-truncation-bug.md`
**Status:** Fixed

---

## Root Cause Analysis

After thorough code analysis, we found that the **LLM (Eino ReAct agent) receives the full, untruncated tool result**. The 500-character truncation was only applied to the SSE/WS display layer — the events streamed to web clients for UI rendering.

### What was happening

```
Tool execution → output.Response (6027 bytes, full)
       ↓
Eino agent internal loop → Tool message with FULL result → LLM sees everything ✓
       ↓
Callback (OnToolEnd) → event.Content = preview (500 chars), metadata["full_result"] = full
       ↓
SSE event → sent only the 500-char preview as "content" ← THIS was the bug
```

The log line `content_length=503` refers to the preview stored in `event.Content`, not what the LLM receives. The LLM processes the complete `output.Response` through Eino's internal message graph.

### UUID Hallucination

The corrupted UUIDs (`9c11-9c11-9c11...`) are likely caused by model quality, not data truncation. The model had access to all 5 device records with complete UUIDs. We recommend:
- Using a model with stronger JSON reasoning (e.g., GPT-4o, Claude Sonnet)
- If using a smaller model, consider reducing the number of fields in tool responses

## Fix Applied

The SSE/WS events now include the **full tool result**:

### SSE event format (changed)

```json
{
  "tool": "device.list",
  "call_id": "server-device.list-1",
  "content": "<FULL TOOL RESULT>",
  "summary": "5 devices",
  "has_error": false
}
```

**Changes:**
- `content` — now contains the **full, untruncated** tool result (was: 500-char preview)
- `summary` — **new field**, contains a short summary for UI display (e.g., "5 devices", "3 citations")

### WebSocket event format (changed)

```json
{
  "type": "ToolExecutionCompleted",
  "call_id": "server-device.list-1",
  "tool_name": "device.list",
  "result": "<FULL TOOL RESULT>",
  "result_summary": "5 devices",
  "has_error": false,
  "agent_id": "supervisor"
}
```

**Changes:**
- `result` — **new field**, contains the full tool result
- `result_summary` — unchanged, short summary for UI

## Migration Notes for Chirp

1. **SSE clients** reading `content` from `tool_result` events will now receive the full result instead of a truncated preview
2. If your UI was displaying `content` directly, you may want to use `summary` for display and `content` for detailed views
3. **Your workarounds** (compact JSON, device_name parameter, lookupByPartialID) are no longer necessary for the truncation issue, but may still be useful for reducing response sizes

## Workarounds You Can Remove

- `lookupByPartialID` — no longer needed if UUID truncation was the only reason
- Compact JSON responses — you can include full device data if the model handles it well
- `device_name` parameter — still useful as a convenience, but not required as a workaround
