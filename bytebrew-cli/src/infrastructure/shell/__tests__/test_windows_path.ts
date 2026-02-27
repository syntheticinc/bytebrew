import { ShellSession } from '../ShellSession.js';

async function test() {
  const session = new ShellSession({ cwd: 'C:/Users/busul' });

  // Test 1: Forward slashes (should work)
  console.log('Test 1: cd with forward slashes');
  const r1 = await session.execute('cd C:/Users/busul/GolandProjects/test-project/vector && pwd', 5000);
  console.log('  completed:', r1.completed, 'exit:', r1.exitCode);
  console.log('  output:', r1.stdout.trim());

  // Test 2: Single backslashes — this is what the LLM sends via gRPC
  // In JS source, \\ produces single backslash in the string
  const backslashCmd = 'cd C:\\Users\\busul\\GolandProjects\\test-project\\vector && pwd';
  console.log('\nTest 2: cd with single backslashes');
  console.log('  command chars:', [...backslashCmd.slice(0, 20)].map(c => c.charCodeAt(0)));
  const r2 = await session.execute(backslashCmd, 5000);
  console.log('  completed:', r2.completed, 'exit:', r2.exitCode);
  console.log('  output:', r2.stdout.trim());

  // Test 3: go build in the directory (if cd works)
  console.log('\nTest 3: go version (quick command)');
  const r3 = await session.execute('go version', 5000);
  console.log('  completed:', r3.completed, 'exit:', r3.exitCode);
  console.log('  output:', r3.stdout.trim());

  session.destroy();
  console.log('\nDone!');
}

test().catch(e => { console.error('ERROR:', e); process.exit(1); });
