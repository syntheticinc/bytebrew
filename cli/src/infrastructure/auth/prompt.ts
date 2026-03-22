import * as readline from 'readline';

export function prompt(question: string): Promise<string> {
  const rl = readline.createInterface({ input: process.stdin, output: process.stderr });
  return new Promise((resolve) => {
    rl.question(question, (answer) => {
      rl.close();
      resolve(answer);
    });
  });
}

export function promptPassword(question: string): Promise<string> {
  return new Promise((resolve) => {
    process.stderr.write(question);

    const stdin = process.stdin;
    const wasRaw = stdin.isRaw;
    if (stdin.isTTY) stdin.setRawMode(true);
    stdin.resume();
    stdin.setEncoding('utf8');

    let password = '';

    const onData = (ch: string) => {
      const code = ch.charCodeAt(0);

      // Enter
      if (ch === '\r' || ch === '\n') {
        stdin.removeListener('data', onData);
        if (stdin.isTTY) stdin.setRawMode(wasRaw ?? false);
        stdin.pause();
        process.stderr.write('\n');
        resolve(password);
        return;
      }

      // Ctrl+C
      if (code === 3) {
        stdin.removeListener('data', onData);
        if (stdin.isTTY) stdin.setRawMode(wasRaw ?? false);
        process.stderr.write('\n');
        process.exit(1);
      }

      // Backspace / Delete
      if (code === 127 || code === 8) {
        if (password.length > 0) {
          password = password.slice(0, -1);
          process.stderr.write('\b \b');
        }
        return;
      }

      // Ignore other control characters
      if (code < 32) return;

      password += ch;
      process.stderr.write('*');
    };

    stdin.on('data', onData);
  });
}
