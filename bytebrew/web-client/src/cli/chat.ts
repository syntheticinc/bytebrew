// REST API SSE chat client for testing
// Usage: bun run chat -- "your message" [--agent supervisor] [--server http://localhost:9443]

const args = process.argv.slice(2);
let message = "hello";
let agent = "supervisor";
let server = "http://localhost:9443";
let user = "admin";
let password = "testpass123";

// Parse args
for (let i = 0; i < args.length; i++) {
  if (args[i] === "--agent") agent = args[++i];
  else if (args[i] === "--server") server = args[++i];
  else if (args[i] === "--user") user = args[++i];
  else if (args[i] === "--password") password = args[++i];
  else if (!args[i].startsWith("--")) message = args[i];
}

async function main() {
  // 1. Login
  console.log(`Logging in to ${server}...`);
  const loginResp = await fetch(`${server}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username: user, password }),
  });

  if (!loginResp.ok) {
    console.error(`Login failed: ${loginResp.status} ${await loginResp.text()}`);
    process.exit(1);
  }

  const { token } = (await loginResp.json()) as { token: string };
  console.log("Authenticated.\n");

  // 2. Send chat message (SSE)
  console.log(`Agent: ${agent}`);
  console.log(`Message: ${message}`);
  console.log("---");

  const chatResp = await fetch(`${server}/api/v1/agents/${agent}/chat`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
      Accept: "text/event-stream",
    },
    body: JSON.stringify({ message, user_id: "test-client" }),
  });

  if (!chatResp.ok) {
    console.error(`Error: ${chatResp.status} ${await chatResp.text()}`);
    process.exit(1);
  }

  // 3. Stream SSE events
  const reader = chatResp.body?.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let fullMessage = "";

  while (reader) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split("\n");
    buffer = lines.pop() || "";

    let eventType = "";
    for (const line of lines) {
      if (line.startsWith("event: ")) {
        eventType = line.slice(7).trim();
      } else if (line.startsWith("data: ")) {
        const data = line.slice(6);
        try {
          const parsed = JSON.parse(data);
          switch (eventType) {
            case "thinking":
              process.stdout.write(`[thinking] ${parsed.content}`);
              break;
            case "message":
              process.stdout.write(parsed.content);
              fullMessage += parsed.content;
              break;
            case "tool_call":
              console.log(`\n[tool] ${parsed.tool}: ${parsed.content}`);
              break;
            case "tool_result":
              console.log(
                `[result] ${parsed.tool} -> ${parsed.content?.substring(0, 100)}`
              );
              break;
            case "done":
              console.log(`\n\n[done] ${parsed.status}`);
              break;
            case "error":
              console.error(`\n[error] ${parsed.message}`);
              break;
          }
        } catch {
          // ignore parse errors for non-JSON data lines
        }
      }
    }
  }

  console.log("\n---");
  if (fullMessage) {
    console.log(`Full response: ${fullMessage}`);
  }
}

main().catch(console.error);
