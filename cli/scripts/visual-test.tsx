import React from 'react';
import { render } from 'ink-testing-library';
import { ChatApp } from '../src/presentation/app/ChatApp.js';
import { createContainer } from '../src/config/container.js';
import { useViewStore } from '../src/presentation/store/viewStore.js';
import path from 'path';

const tick = (ms = 100) => new Promise<void>(r => setTimeout(r, ms));

async function waitFor(predicate: () => boolean, timeoutMs: number): Promise<void> {
  const start = Date.now();
  while (!predicate()) {
    if (Date.now() - start > timeoutMs) throw new Error('waitFor timeout');
    await tick();
  }
}

async function main() {
  const testProjectRoot = path.resolve(import.meta.dir, '../../test-project');

  const container = createContainer({
    projectRoot: testProjectRoot,
    serverAddress: 'localhost:60401',
    projectKey: 'visual-test',
  });

  console.log('Starting visual test...');
  console.log(`Project root: ${testProjectRoot}`);
  console.log('Rendering ChatApp with initial question...\n');

  const { lastFrame, stdin, unmount } = render(
    <ChatApp container={container} initialQuestion="создай файл hello.go с функцией Hello()" />
  );

  // Wait for connection
  console.log('Waiting for connection...');
  try {
    await waitFor(() => useViewStore.getState().connectionStatus === 'connected', 10000);
    console.log('Connected!\n');
  } catch {
    console.error('Failed to connect within 10 seconds');
    console.log('Current status:', useViewStore.getState().connectionStatus);
    console.log('\nFrame:');
    console.log(lastFrame());
    unmount();
    await container.dispose();
    process.exit(1);
  }

  // Wait for processing to start
  await tick(1000);

  // Wait for processing to complete (max 2 min)
  console.log('Waiting for processing to complete...');
  try {
    await waitFor(() => !useViewStore.getState().isProcessing, 120000);
    console.log('Processing complete!\n');
  } catch {
    console.error('Processing did not complete within 2 minutes');
  }

  // Extra ticks for final render
  await tick(500);

  // Capture supervisor view
  console.log('='.repeat(60));
  console.log('=== SUPERVISOR VIEW ===');
  console.log('='.repeat(60));
  console.log(lastFrame());

  // Try switching tabs
  stdin.write('\x1b[Z'); // Shift+Tab
  await tick(200);

  console.log('\n' + '='.repeat(60));
  console.log('=== AFTER TAB SWITCH (Agent View) ===');
  console.log('='.repeat(60));
  console.log(lastFrame());

  // Switch again
  stdin.write('\x1b[Z');
  await tick(200);

  console.log('\n' + '='.repeat(60));
  console.log('=== AFTER 2nd TAB SWITCH (Back to Supervisor) ===');
  console.log('='.repeat(60));
  console.log(lastFrame());

  // Show agent info
  const agents = container.agentStateManager.getAllAgents();
  console.log('\n--- Agent State ---');
  for (const agent of agents) {
    console.log(`  ${agent.agentId}: role=${agent.role}, status=${agent.completionStatus}, task="${agent.taskDescription}"`);
  }

  // Cleanup
  unmount();
  await container.dispose();
  console.log('\nVisual test complete.');
  process.exit(0);
}

main().catch(err => {
  console.error('Visual test failed:', err);
  process.exit(1);
});
