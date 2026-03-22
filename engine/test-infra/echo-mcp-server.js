#!/usr/bin/env node
// Minimal MCP echo server for testing (stdio transport, JSON-RPC 2.0)
const readline = require('readline');
const rl = readline.createInterface({ input: process.stdin });

rl.on('line', (line) => {
  try {
    const req = JSON.parse(line);
    let resp;

    if (req.method === 'initialize') {
      resp = { jsonrpc: '2.0', id: req.id, result: { protocolVersion: '2024-11-05', capabilities: { tools: {} }, serverInfo: { name: 'echo-mcp', version: '1.0.0' } } };
    } else if (req.method === 'notifications/initialized') {
      return; // notification, no response
    } else if (req.method === 'tools/list') {
      resp = { jsonrpc: '2.0', id: req.id, result: { tools: [{ name: 'echo', description: 'Echo back the input message', inputSchema: { type: 'object', properties: { message: { type: 'string', description: 'Message to echo' } }, required: ['message'] } }] } };
    } else if (req.method === 'tools/call') {
      const msg = req.params?.arguments?.message || 'no message';
      resp = { jsonrpc: '2.0', id: req.id, result: { content: [{ type: 'text', text: 'Echo: ' + msg }] } };
    } else {
      resp = { jsonrpc: '2.0', id: req.id, error: { code: -32601, message: 'Method not found' } };
    }

    process.stdout.write(JSON.stringify(resp) + '\n');
  } catch (e) {
    // ignore parse errors
  }
});
