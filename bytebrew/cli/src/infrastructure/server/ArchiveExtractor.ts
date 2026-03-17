import path from "path";
import fs from "fs/promises";

export async function extractTarGz(
  buffer: Buffer,
  destDir: string,
  binaryName: string,
): Promise<void> {
  // Write archive to temp file, then extract using system tar
  const tmpArchive = path.join(destDir, `_download_${Date.now()}.tar.gz`);
  try {
    await fs.writeFile(tmpArchive, buffer);
    const proc = Bun.spawn(["tar", "-xzf", tmpArchive, "-C", destDir], {
      stdout: "pipe",
      stderr: "pipe",
    });
    const exited = await proc.exited;
    if (exited !== 0) {
      const stderr = await new Response(proc.stderr).text();
      throw new Error(`tar extract failed: ${stderr}`);
    }

    // Try to find and move the binary to the top level
    await promoteBinary(destDir, binaryName);
  } finally {
    await fs.rm(tmpArchive, { force: true });
  }
}

export async function extractZip(
  buffer: Buffer,
  destDir: string,
  binaryName: string,
): Promise<void> {
  const tmpArchive = path.join(destDir, `_download_${Date.now()}.zip`);
  try {
    await fs.writeFile(tmpArchive, buffer);

    // Use tar on Windows (available since Win10 1803) or unzip on Unix
    const cmd =
      process.platform === "win32"
        ? ["tar", "-xf", tmpArchive, "-C", destDir]
        : ["unzip", "-o", tmpArchive, "-d", destDir];

    const proc = Bun.spawn(cmd, {
      stdout: "pipe",
      stderr: "pipe",
    });
    const exited = await proc.exited;
    if (exited !== 0) {
      const stderr = await new Response(proc.stderr).text();
      throw new Error(`unzip failed: ${stderr}`);
    }

    await promoteBinary(destDir, binaryName);
  } finally {
    await fs.rm(tmpArchive, { force: true });
  }
}

/**
 * After extraction, the binary might be nested in a subdirectory.
 * Find it and copy to the top-level bin dir.
 */
export async function promoteBinary(
  dir: string,
  binaryName: string,
): Promise<void> {
  const ext = process.platform === "win32" ? ".exe" : "";
  const targetName = binaryName + ext;
  const targetPath = path.join(dir, targetName);

  // Already at top level?
  try {
    await fs.access(targetPath);
    return;
  } catch {
    // not at top level, search subdirectories
  }

  // BFS through extracted dirs to find the binary
  const found = await findFileRecursive(dir, targetName);
  if (!found) {
    throw new Error(`Binary "${targetName}" not found in extracted archive`);
  }
  if (found !== targetPath) {
    await fs.copyFile(found, targetPath);
    if (process.platform !== "win32") {
      await fs.chmod(targetPath, 0o755);
    }
  }
}

export async function findFileRecursive(
  dir: string,
  name: string,
): Promise<string | undefined> {
  const entries = await fs.readdir(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isFile() && entry.name === name) {
      return fullPath;
    }
    if (entry.isDirectory() && !entry.name.startsWith("_") && !entry.name.startsWith(".")) {
      const found = await findFileRecursive(fullPath, name);
      if (found) return found;
    }
  }
  return undefined;
}
