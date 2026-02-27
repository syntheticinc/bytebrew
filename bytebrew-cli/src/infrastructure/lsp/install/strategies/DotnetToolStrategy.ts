import type { DotnetToolSpec, InstallResult } from "../types.js";
import type { BinDirectory } from "../BinDirectory.js";
import { getLogger } from "../../../../lib/logger.js";

const INSTALL_TIMEOUT_MS = 120_000;

/**
 * Installs .NET-based LSP servers via `dotnet tool install <package>`.
 * Uses --tool-path to direct the binary into the managed bin directory.
 */
export async function installDotnetTool(
  spec: DotnetToolSpec,
  binDir: BinDirectory,
): Promise<InstallResult> {
  const logger = getLogger();

  const dotnetBin = Bun.which("dotnet");
  if (!dotnetBin) {
    return { success: false, error: "dotnet not found in PATH" };
  }

  await binDir.ensureExists();

  logger.info(`[LSP] dotnet: installing ${spec.package}`, { toolPath: binDir.getPath() });

  try {
    const proc = Bun.spawn(
      [dotnetBin, "tool", "install", spec.package, "--tool-path", binDir.getPath()],
      {
        stdout: "pipe",
        stderr: "pipe",
      },
    );

    const exited = await Promise.race([
      proc.exited,
      sleep(INSTALL_TIMEOUT_MS).then(() => {
        proc.kill();
        throw new Error(`dotnet tool install timed out after ${INSTALL_TIMEOUT_MS}ms`);
      }),
    ]);

    if (exited !== 0) {
      const stderr = await new Response(proc.stderr).text();
      // dotnet tool install returns exit code 1 if already installed
      if (stderr.includes("already installed")) {
        return { success: true, binaryPath: binDir.binaryPath(spec.package) };
      }
      return { success: false, error: `dotnet tool install ${spec.package} failed (exit ${exited}): ${stderr.slice(0, 500)}` };
    }

    return { success: true, binaryPath: binDir.binaryPath(spec.package) };
  } catch (err) {
    return { success: false, error: String(err) };
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
