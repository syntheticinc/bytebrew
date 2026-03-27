# ByteBrew — Response to SSE Streaming Bug Report

**From:** ByteBrew Engineering
**Date:** 2026-03-27
**In response to:** `bytebrew-sse-streaming-bug.md`

---

## Status: FIXED

This issue was identified and fixed on 2026-03-26. The fix is included in `bytebrew/engine:latest` on Docker Hub.

## Root Cause

Go's `net/http` buffers small responses and automatically sets `Content-Length`. For SSE streaming, this causes the entire response to be sent as a single batch instead of event-by-event.

## Fix Applied

Commit `b33cb035` — calls `w.WriteHeader(http.StatusOK)` before writing the first event. This commits the response headers immediately, forcing Go to use chunked transfer encoding.

Additionally, `http.NewResponseController(w)` is used instead of `w.(http.Flusher)` to ensure `Flush()` works through middleware wrappers (chi router wraps ResponseWriter).

## Verification

After `docker pull bytebrew/engine:latest`:

```bash
curl -v -N -X POST http://localhost:8443/api/v1/agents/kilo-assistant/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message":"hello"}'
```

**Expected response headers (no Content-Length):**
```
HTTP/1.1 200 OK
Cache-Control: no-cache
Connection: keep-alive
Content-Type: text/event-stream
X-Accel-Buffering: no
```

**Expected streaming behavior:**
- Each `message_delta` event flushed immediately
- No `Content-Length` header
- Events arrive in real-time as LLM generates tokens

## Action Required

```bash
docker pull bytebrew/engine:latest
docker compose down && docker compose up -d
```

If you're pinning a specific version, use `bytebrew/engine:1.0.0` or later.

## Additional Note

If your `ai-assistant-service` proxy sits between the frontend and ByteBrew, ensure it also streams SSE correctly:
- Do NOT buffer the response body
- Forward `Transfer-Encoding: chunked`
- Call `Flush()` after forwarding each SSE event line
- Set `X-Accel-Buffering: no` if behind nginx

---

*ByteBrew Engineering Team*
