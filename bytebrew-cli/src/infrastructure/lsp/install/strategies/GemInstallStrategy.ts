import type { GemInstallSpec, InstallResult } from "../types.js";
import type { BinDirectory } from "../BinDirectory.js";
import { getLogger } from "../../../../lib/logger.js";

const INSTALL_TIMEOUT_MS = 120_000;

/**
 * Installs Ruby-based LSP servers via `gem install <package>`.
 * Uses --bindir to direct the binary into the managed bin directory.
 */
export async function installGemPackage(
  spec: GemInstallSpec,
  binDir: BinDirectory,
): Promise<InstallResult> {
  const logger = getLogger();

  const gemBin = Bun.which("gem");
  if (!gemBin) {
    return { success: false, error: "gem not found in PATH" };
  }

  await binDir.ensureExists();

  logger.info(`[LSP] gem: installing ${spec.package}`, { bindir: binDir.getPath() });

  try {
    const proc = Bun.spawn(
      [gemBin, "install", spec.package, "--bindir", binDir.getPath(), "--no-document"],
      {
        stdout: "pipe",
        stderr: "pipe",
      },
    );

    const exited = await Promise.race([
      proc.exited,
      sleep(INSTALL_TIMEOUT_MS).then(() => {
        proc.kill();
        throw new Error(`gem install timed out after ${INSTALL_TIMEOUT_MS}ms`);
      }),
    ]);

    if (exited !== 0) {
      const stderr = await new Response(proc.stderr).text();
      return { success: false, error: `gem install ${spec.package} failed (exit ${exited}): ${stderr.slice(0, 500)}` };
    }

    return { success: true, binaryPath: binDir.binaryPath(spec.package) };
  } catch (err) {
    return { success: false, error: String(err) };
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
