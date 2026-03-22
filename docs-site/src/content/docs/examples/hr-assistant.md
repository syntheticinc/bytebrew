---
title: "HR Assistant"
description: "Knowledge Base (RAG), structured Q&A, and escalation — build an HR chatbot that searches company policies and handles leave requests."
---

An HR Assistant that answers policy questions using your company knowledge base, handles leave requests through structured Q&A, and escalates complex cases to a human.

## What this demonstrates

| Engine Feature | How it's used |
|---|---|
| **Knowledge Base (RAG)** | Searches company policies, handbooks, and FAQs to ground answers in real documents |
| **`ask_user` structured Q&A** | Collects leave request details step-by-step (dates, type, reason) |
| **Escalation webhook** | Routes unresolvable or sensitive cases to a human HR representative |
| **MCP tools** | Connects to your HRIS to look up employee data and submit requests |

## Architecture

```
User
 │
 ▼
┌─────────────────────────────────┐
│         HR Assistant Agent      │
│                                 │
│  Tools:                         │
│  ├── knowledge_search (RAG)     │
│  ├── ask_user (structured Q&A)  │
│  ├── get_employee (MCP)         │
│  ├── get_leave_balance (MCP)    │
│  ├── submit_leave_request (MCP) │
│  └── escalate (webhook)         │
└─────────────────────────────────┘
         │            │
         ▼            ▼
   Knowledge Base    HRIS API
   (policies, FAQ)   (employees, leave)
```

## Quick start

```bash
git clone https://github.com/syntheticinc/bytebrew-examples
cd bytebrew-examples/hr-assistant
docker compose up
```

Open the Admin Dashboard at `http://localhost:8080/admin` to upload your policy documents to the Knowledge Base, then start chatting at `http://localhost:8080`.

## Agent configuration

```yaml
# agents.yaml
agents:
  - name: hr-assistant
    model: gpt-4o
    system_prompt: |
      You are an HR assistant for {{company_name}}.
      Answer policy questions using the knowledge base.
      For leave requests, collect all required information before submitting.
      Escalate to a human if you cannot resolve the issue or it involves
      sensitive topics (termination, complaints, salary disputes).
    tools:
      - knowledge_search
      - ask_user
      - escalate
    mcp_servers:
      - hris
    settings:
      company_name: "Acme Corp"

mcp_servers:
  - name: hris
    url: http://hris-api:3000/mcp
    tools:
      - get_employee
      - get_leave_balance
      - submit_leave_request
```

## Example conversations

### Policy question (RAG)

The agent searches the knowledge base and returns a grounded answer:

```
User: What's the policy on remote work?

SSE events:
  event: tool_call
  data: {"tool": "knowledge_search", "input": {"query": "remote work policy"}}

  event: tool_result
  data: {"results": [{"title": "Remote Work Policy v2.3", "content": "Employees may work remotely up to 3 days per week with manager approval..."}]}

  event: message
  data: {"content": "According to our Remote Work Policy (v2.3), you may work remotely up to 3 days per week with your manager's approval. You'll need to submit a request through the HR portal at least 48 hours in advance..."}
```

### Leave request (structured Q&A)

The agent collects details step-by-step using `ask_user`:

```
User: I'd like to request time off.

SSE events:
  event: tool_call
  data: {"tool": "get_leave_balance", "input": {"employee_id": "emp-442"}}

  event: tool_result
  data: {"vacation": 12, "sick": 5, "personal": 3}

  event: tool_call
  data: {"tool": "ask_user", "input": {"question": "What type of leave? (vacation / sick / personal)\nYou have: 12 vacation, 5 sick, 3 personal days remaining."}}

  event: user_input
  data: {"response": "vacation"}

  event: tool_call
  data: {"tool": "ask_user", "input": {"question": "What dates? (start — end)"}}

  event: user_input
  data: {"response": "April 7–11"}

  event: tool_call
  data: {"tool": "submit_leave_request", "input": {"employee_id": "emp-442", "type": "vacation", "start": "2026-04-07", "end": "2026-04-11"}}

  event: tool_result
  data: {"request_id": "lr-891", "status": "pending_approval"}

  event: message
  data: {"content": "Your vacation request (April 7–11, 5 days) has been submitted. Request ID: lr-891. Your manager will receive a notification for approval."}
```

### Escalation

When the agent cannot resolve an issue, it escalates:

```
User: I want to file a complaint about my manager.

SSE events:
  event: tool_call
  data: {"tool": "escalate", "input": {"reason": "Employee complaint about manager — requires human HR intervention", "priority": "high"}}

  event: message
  data: {"content": "I understand this is a sensitive matter. I've escalated your case to an HR representative who will contact you within 24 hours. Your case reference is ESC-2847."}
```

## Run it yourself

Full source code with Docker Compose, sample policy documents, and a mock HRIS API:

[github.com/syntheticinc/bytebrew-examples/hr-assistant](https://github.com/syntheticinc/bytebrew-examples/tree/main/hr-assistant)
