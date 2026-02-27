import type { InstallSpec, InstallResult } from "./types.js";
import { BinDirectory } from "./BinDirectory.js";
import { installNpmPackage } from "./strategies/NpmInstallStrategy.js";
import { installGoPackage } from "./strategies/GoInstallStrategy.js";
import { installFromGithubRelease } from "./strategies/GithubReleaseStrategy.js";
import { installGemPackage } from "./strategies/GemInstallStrategy.js";
import { installDotnetTool } from "./strategies/DotnetToolStrategy.js";
import { getLogger } from "../../../lib/logger.js";

/**
 * Orchestrates LSP server binary installation.
 * Delegates to strategy implementations based on InstallSpec.type.
 * Coalesces concurrent install requests for the same server.
 */
export class LspInstaller {
  private readonly binDir: BinDirectory;
  private readonly installing = new Map<string, Promise<InstallResult>>();

  constructor(binDir?: BinDirectory) {
    this.binDir = binDir ?? new BinDirectory();
  }

  isDisabled(): boolean {
    return process.env.BYTEBREW_DISABLE_LSP_DOWNLOAD === "true";
  }

  getBinDir(): BinDirectory {
    return this.binDir;
  }

  /**
   * Attempt to install a server binary.
   * Concurrent calls for same serverId are coalesced.
   */
  async install(serverId: string, spec: InstallSpec): Promise<InstallResult> {
    if (this.isDisabled()) {
      return { success: false, error: "Auto-install disabled (BYTEBREW_DISABLE_LSP_DOWNLOAD=true)" };
    }

    const existing = this.installing.get(serverId);
    if (existing) return existing;

    const task = this.doInstall(serverId, spec);
    this.installing.set(serverId, task);
    task.finally(() => {
      if (this.installing.get(serverId) === task) {
        this.installing.delete(serverId);
      }
    });

    return task;
  }

  private async doInstall(
    serverId: string,
    spec: InstallSpec,
  ): Promise<InstallResult> {
    const logger = getLogger();
    logger.info(`[LSP] installing ${serverId}`, { type: spec.type });

    try {
      const result = await this.dispatch(spec);
      if (result.success) {
        logger.info(`[LSP] installed ${serverId}`, { path: result.binaryPath });
      } else {
        logger.error(`[LSP] install failed for ${serverId}`, { error: result.error });
      }
      return result;
    } catch (err) {
      const error = err instanceof Error ? err.message : String(err);
      logger.error(`[LSP] install crashed for ${serverId}`, { error });
      return { success: false, error };
    }
  }

  private dispatch(spec: InstallSpec): Promise<InstallResult> {
    switch (spec.type) {
      case "npm":
        return installNpmPackage(spec, this.binDir);
      case "go":
        return installGoPackage(spec, this.binDir);
      case "github-release":
        return installFromGithubRelease(spec, this.binDir);
      case "gem":
        return installGemPackage(spec, this.binDir);
      case "dotnet":
        return installDotnetTool(spec, this.binDir);
    }
  }
}
