---
title: Knowledge / RAG
description: Retrieval-Augmented Generation — ground agent responses in your documents with automatic indexing and vector search.
---

Retrieval-Augmented Generation (RAG) lets agents answer questions based on your documents. Instead of relying solely on the LLM's training data, the agent searches a knowledge base and includes relevant passages in its context before generating a response.

## How it works

- Set `knowledge: "./path/"` in agent config to enable RAG for that agent.
- The engine auto-indexes all documents in the folder at startup (Markdown, TXT, PDF, HTML).
- A `knowledge_search` tool is injected automatically when a knowledge path is configured.
- When the agent calls `knowledge_search`, the engine performs a vector similarity search and returns the most relevant passages.
- The agent uses these passages to generate grounded, accurate responses.

```
# Document indexing flow:
#
#   ./docs/support/
#   |-- faq.md              --> chunked, embedded, indexed
#   |-- returns-policy.txt  --> chunked, embedded, indexed
#   |-- product-guide.pdf   --> extracted, chunked, embedded, indexed
#   |-- setup.html          --> parsed, chunked, embedded, indexed
#
#   Agent calls knowledge_search("return policy for electronics")
#   --> Engine finds the most relevant chunks from returns-policy.txt
#   --> Agent uses them to answer: "Our electronics return policy..."
```

## Configuration

```yaml
agents:
  support-bot:
    model: glm-5
    knowledge: "./docs/support/"     # Path to knowledge base folder
    tools:
      - knowledge_search             # Injected automatically, but explicit is fine
    system: |
      Answer customer questions using the knowledge base.
      Always cite which document you found the information in.
      If you cannot find the answer, say so honestly -- do not
      make up information.
```

:::note[Supported file formats]
The engine indexes `.md`, `.txt`, `.pdf`, and `.html` files. Place files in the knowledge folder and restart the engine (or trigger a hot-reload) to index them. Sub-folders are included recursively.
:::

## Per-agent isolation

Each agent has its own isolated knowledge base. Agent A cannot search agent B's documents. This is important for multi-tenant deployments and role-based access:

```yaml
agents:
  sales-bot:
    knowledge: "./docs/sales/"         # Only sales materials
    tools: [knowledge_search]

  hr-bot:
    knowledge: "./docs/hr-policies/"   # Only HR policies
    tools: [knowledge_search]

# sales-bot cannot see HR policies
# hr-bot cannot see sales materials
```

## Best practices

- **Keep documents focused** -- smaller, topic-specific documents work better than large monolithic ones.
- **Use clear headings** -- Markdown headings help the chunking algorithm split documents at logical boundaries.
- **Update regularly** -- keep knowledge bases current. Outdated information leads to incorrect agent responses.
- **Tell the agent to cite sources** -- add instructions in the system prompt to reference which document the answer came from.
- **Set honest boundaries** -- instruct the agent to say "I don't know" rather than hallucinate when the knowledge base does not contain the answer.

---

## What's next

- [Agents & Lifecycle](/concepts/agents/)
- [Tools](/concepts/tools/)
- [Example: Support Agent](/examples/support-agent/)
