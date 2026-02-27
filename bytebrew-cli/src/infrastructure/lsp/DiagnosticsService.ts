import path from "path";
import type { Diagnostic } from "vscode-languageserver-types";
import { LspManager } from "./LspManager.js";
import { getLogger } from "../../lib/logger.js";

const MAX_DIAGNOSTICS_PER_FILE = 20;
const MAX_PROJECT_DIAGNOSTICS_FILES = 5;

export class DiagnosticsService {
  private manager: LspManager;

  constructor(manager: LspManager) {
    this.manager = manager;
  }

  /**
   * Run diagnostics after a file write/edit.
   * Returns formatted diagnostics string for the LLM, or empty string if none.
   * BEST-EFFORT: errors don't break tool execution.
   */
  async runAfterWrite(filePath: string): Promise<string> {
    try {
      await this.manager.touchFile(filePath, true);
      const allDiags = await this.manager.diagnostics();
      return this.formatForLLM(filePath, allDiags);
    } catch (err) {
      const logger = getLogger();
      logger.error("[LSP] diagnostics failed", { error: err, file: filePath });
      return "";
    }
  }

  /**
   * Format diagnostics for LLM consumption.
   * Mirrors OpenCode write.ts format:
   * - Only ERROR severity (1)
   * - Max 20 errors per file
   * - Max 5 other files shown
   * - <diagnostics> XML tags
   */
  private formatForLLM(
    filePath: string,
    diagnostics: Record<string, Diagnostic[]>,
  ): string {
    const normalizedPath = path.normalize(filePath);
    let output = "";
    let projectDiagnosticsCount = 0;

    for (const [file, issues] of Object.entries(diagnostics)) {
      const errors = issues.filter((item) => item.severity === 1);
      if (errors.length === 0) continue;

      const limited = errors.slice(0, MAX_DIAGNOSTICS_PER_FILE);
      const suffix =
        errors.length > MAX_DIAGNOSTICS_PER_FILE
          ? `\n... and ${errors.length - MAX_DIAGNOSTICS_PER_FILE} more`
          : "";

      const formatted = limited.map((d) => prettyDiagnostic(d)).join("\n");

      if (path.normalize(file) === normalizedPath) {
        output += `\n\nLSP errors detected in this file, please fix:\n<diagnostics file="${filePath}">\n${formatted}${suffix}\n</diagnostics>`;
        continue;
      }

      if (projectDiagnosticsCount >= MAX_PROJECT_DIAGNOSTICS_FILES) continue;
      projectDiagnosticsCount++;
      output += `\n\nLSP errors detected in other files:\n<diagnostics file="${file}">\n${formatted}${suffix}\n</diagnostics>`;
    }

    return output;
  }

  async warmup(): Promise<void> {
    try {
      await this.manager.warmup();
    } catch (err) {
      const logger = getLogger();
      logger.error("[LSP] warmup failed", { error: err });
    }
  }

  async dispose(): Promise<void> {
    await this.manager.dispose();
  }
}

function prettyDiagnostic(diagnostic: Diagnostic): string {
  const severityMap: Record<number, string> = {
    1: "ERROR",
    2: "WARN",
    3: "INFO",
    4: "HINT",
  };
  const severity = severityMap[diagnostic.severity || 1] || "ERROR";
  const line = diagnostic.range.start.line + 1;
  const col = diagnostic.range.start.character + 1;
  return `${severity} [${line}:${col}] ${diagnostic.message}`;
}
