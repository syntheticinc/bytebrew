import type { GoInstallSpec, InstallResult } from "../types.js";
import type { BinDirectory } from "../BinDirectory.js";
import { getLogger } from "../../../../lib/logger.js";

const INSTALL_TIMEOUT_MS = 120_000;

/**
 * Installs Go-based LSP servers via `go install <module>@latest`.
 * Uses GOBIN to direct the binary into the managed bin directory.
 */
export async function installGoPackage(
  spec: GoInstallSpec,
  binDir: BinDirectory,
): Promise<InstallResult> {
  const logger = getLogger();

  const goBin = Bun.which("go");
  if (!goBin) {
    return { success: false, error: "go not found in PATH" };
  }

  await binDir.ensureExists();
  const version = spec.version || "@latest";
  const target = `${spec.module}${version}`;

  logger.info(`[LSP] go: installing ${target}`, { GOBIN: binDir.getPath() });

  try {
    const proc = Bun.spawn([goBin, "install", target], {
      stdout: "pipe",
      stderr: "pipe",
      env: { ...process.env, GOBIN: binDir.getPath() },
    });

    const exited = await Promise.race([
      proc.exited,
      sleep(INSTALL_TIMEOUT_MS).then(() => {
        proc.kill();
        throw new Error(`go install timed out after ${INSTALL_TIMEOUT_MS}ms`);
      }),
    ]);

    if (exited !== 0) {
      const stderr = await new Response(proc.stderr).text();
      return { success: false, error: `go install ${target} failed (exit ${exited}): ${stderr.slice(0, 500)}` };
    }

    // Extract binary name from module path (last segment)
    const parts = spec.module.split("/");
    const binaryName = parts[parts.length - 1];
    return { success: true, binaryPath: binDir.binaryPath(binaryName) };
  } catch (err) {
    return { success: false, error: String(err) };
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
