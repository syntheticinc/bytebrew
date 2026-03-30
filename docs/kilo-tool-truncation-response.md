# Response: Tool Result Truncation Bug

**Date:** 2026-03-30
**In response to:** `bytebrew-tool-result-truncation-bug.md`, `bytebrew-sse-truncation-followup.md`
**Status:** Fixed (deployed 2026-03-30T12:38Z)

---

## Root Cause Analysis

Two issues found:

### Issue 1: SSE content truncated to 500 chars

The LLM (Eino ReAct agent) always received the **full, untruncated tool result**. The 500-character truncation was only in the SSE/WS display layer — the events streamed to web clients.

```
Tool execution → output.Response (6027 bytes, full)
       ↓
Eino agent internal loop → Tool message with FULL result → LLM sees everything ✓
       ↓
Callback (OnToolEnd) → event.Content = preview (500 chars), metadata["full_result"] = full
       ↓
SSE event → sent only the 500-char preview as "content" ← THIS was the bug
```

The log `content_length=503` refers to the preview stored in `event.Content` (domain event), not what the LLM receives or what SSE now sends. After the fix, SSE `content` contains the full result.

**Fix:** SSE/WS events now include the full tool result in `content`.

### Issue 2: Initial deploy didn't update the running container

After the first fix was pushed (commit `bebfc957`), the CI pipeline built the correct Docker image and pushed it to Docker Hub, but the VPS container was **not recreated** because:
1. `docker compose up -d` doesn't recreate containers when the tag (`latest`) is already present locally
2. The `deploy` user didn't have Docker socket permissions

**Fix:**
- CI now uses `docker compose up -d --force-recreate` (commit `9991a329`)
- `deploy` user added to `docker` group on VPS
- Verified: CI now correctly recreates containers on every deploy

### Text splitting between tool calls (NOT a bug)

The observed behavior:
```
"его устройства Dragino LDS02."    ← text block 1
[Tools: device.list]                ← tool call
"Общая информация об устройствах"  ← text block 2
```

This is **expected ReAct agent behavior**, not a bug. The LLM generates text, then decides to call a tool. After receiving the tool result, it generates more text. These are separate steps in the ReAct loop:

1. **Step 1:** LLM outputs "...Dragino LDS02." + tool_call(device.list)
2. Tool executes, result returned to LLM
3. **Step 2:** LLM outputs "Общая информация..." based on tool result

In SSE this appears as: `message_delta` events → `tool_call` → `tool_result` → `message_delta` events

**Recommendation:** On the frontend side, group text blocks around tool calls visually. For example, show tool call as an expandable card between text paragraphs.

## SSE Event Format (current)

```json
{
  "tool": "device.list",
  "call_id": "server-device.list-1",
  "content": "<FULL TOOL RESULT>",
  "summary": "5 devices",
  "has_error": false
}
```

- `content` — **full, untruncated** tool result
- `summary` — short description for UI (e.g., "5 devices", "3 citations")

## WebSocket Event Format (current)

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

## Action Required (Chirp side)

**Pull the latest image:**
```bash
docker pull bytebrew/engine:latest
docker compose up -d --force-recreate
```

After updating, verify with:
```bash
# Send a chat that triggers a tool with >500 char result
# Check SSE event: content should now be the full result, not truncated
```

## UUID Hallucination

The corrupted UUIDs (`9c11-9c11-9c11...`) were likely caused by model quality, not data truncation. The model had access to all 5 device records with complete UUIDs. We recommend:
- Using a model with stronger JSON reasoning (e.g., GPT-4o, Claude Sonnet)
- If using a smaller model, consider reducing the number of fields in tool responses

## Workarounds You Can Remove

- `lookupByPartialID` — no longer needed if UUID truncation was the only reason
- Compact JSON responses — you can include full device data if the model handles it well
- `device_name` parameter — still useful as a convenience, but not required as a workaround
