// Quick health check
// Usage: bun run health [server-url]

const server = process.argv[2] || "http://localhost:9443";

async function main() {
  try {
    const resp = await fetch(`${server}/api/v1/health`);
    if (!resp.ok) {
      console.error(`Health check failed: ${resp.status}`);
      process.exit(1);
    }
    console.log(JSON.stringify(await resp.json(), null, 2));
  } catch (err) {
    console.error(`Connection failed: ${err}`);
    process.exit(1);
  }
}

main();
