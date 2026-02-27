import { spawn } from 'child_process';

/** Open a URL in the default browser. Silent on failure. */
export function openBrowser(url: string): void {
  try {
    const [cmd, args]: [string, string[]] =
      process.platform === 'win32' ? ['cmd', ['/c', 'start', '', url]] :
      process.platform === 'darwin' ? ['open', [url]] :
      ['xdg-open', [url]];
    spawn(cmd, args, { detached: true, stdio: 'ignore' }).unref();
  } catch {
    // Silent — fallback URL printed to console by caller
  }
}
