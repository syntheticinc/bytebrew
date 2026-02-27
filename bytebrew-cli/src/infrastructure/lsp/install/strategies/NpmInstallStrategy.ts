import path from "path";
import fs from "fs/promises";
import type { NpmInstallSpec, InstallResult } from "../types.js";
import type { BinDirectory } from "../BinDirectory.js";
import { getLogger } from "../../../../lib/logger.js";

const INSTALL_TIMEOUT_MS = 120_000;

/**
 * Installs Node.js-based LSP servers via `bun install <package>`.
 * Creates a package.json in the bin directory and installs the package there,
 * so the binary appears in <binDir>/node_modules/.bin/.
 */
export async function installNpmPackage(
  spec: NpmInstallSpec,
  binDir: BinDirectory,
): Promise<InstallResult> {
  const logger = getLogger();
  const binPath = binDir.getPath();

  const bunBin = Bun.which("bun");
  if (!bunBin) {
    return { success: false, error: "bun not found in PATH" };
  }

  await binDir.ensureExists();

  // Ensure package.json exists in bin dir
  const pkgJsonPath = path.join(binPath, "package.json");
  try {
    await fs.access(pkgJsonPath);
  } catch {
    await fs.writeFile(pkgJsonPath, JSON.stringify({ private: true, dependencies: {} }, null, 2));
  }

  logger.info(`[LSP] npm: installing ${spec.package}`, { cwd: binPath });

  try {
    const proc = Bun.spawn([bunBin, "add", spec.package], {
      cwd: binPath,
      stdout: "pipe",
      stderr: "pipe",
      env: { ...process.env },
    });

    const exited = await Promise.race([
      proc.exited,
      sleep(INSTALL_TIMEOUT_MS).then(() => {
        proc.kill();
        throw new Error(`bun add timed out after ${INSTALL_TIMEOUT_MS}ms`);
      }),
    ]);

    if (exited !== 0) {
      const stderr = await new Response(proc.stderr).text();
      return { success: false, error: `bun add ${spec.package} failed (exit ${exited}): ${stderr.slice(0, 500)}` };
    }

    // binaryPath is the bin dir — whichBin() will find the actual binary in node_modules/.bin/
    const installed = await binDir.hasBinary(spec.package);
    return { success: true, binaryPath: installed || binPath };
  } catch (err) {
    return { success: false, error: String(err) };
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
