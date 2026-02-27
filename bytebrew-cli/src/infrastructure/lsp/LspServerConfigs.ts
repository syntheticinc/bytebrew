import { spawn, type ChildProcessWithoutNullStreams } from "child_process";
import path from "path";
import os from "os";
import fs from "fs/promises";
import fsSync from "fs";
import type { InstallSpec } from "./install/types.js";

export interface LspServerHandle {
  process: ChildProcessWithoutNullStreams;
  initialization?: Record<string, any>;
}

export interface LspServerConfig {
  id: string;
  extensions: string[];
  /** Optional auto-install specification. If absent, server must be pre-installed. */
  install?: InstallSpec;
  root: (file: string, projectRoot: string) => Promise<string | undefined>;
  spawn: (root: string) => Promise<LspServerHandle | undefined>;
}

// --- Utilities ---

async function fileExists(p: string): Promise<boolean> {
  try {
    await fs.stat(p);
    return true;
  } catch {
    return false;
  }
}

async function findFileUp(
  name: string,
  start: string,
  stop: string,
): Promise<string | undefined> {
  let dir = path.resolve(start);
  const stopResolved = path.resolve(stop);

  while (true) {
    // Handle glob patterns like "*.cabal" or "*.xcodeproj"
    if (name.includes("*")) {
      try {
        const entries = await fs.readdir(dir);
        const pattern = name.replace("*", "");
        const match = entries.find((e) => e.endsWith(pattern));
        if (match) return path.join(dir, match);
      } catch {
        /* ignore */
      }
    } else {
      const candidate = path.join(dir, name);
      if (await fileExists(candidate)) return candidate;
    }

    const parent = path.dirname(dir);
    if (parent === dir) return undefined;
    if (!dir.startsWith(stopResolved)) return undefined;
    dir = parent;
  }
}

function nearestRoot(
  patterns: string[],
  exclude?: string[],
): (file: string, projectRoot: string) => Promise<string | undefined> {
  return async (file, projectRoot) => {
    if (exclude) {
      for (const excl of exclude) {
        const found = await findFileUp(excl, path.dirname(file), projectRoot);
        if (found) return undefined;
      }
    }
    for (const pattern of patterns) {
      const found = await findFileUp(pattern, path.dirname(file), projectRoot);
      if (found) return path.dirname(found);
    }
    return projectRoot;
  };
}

let managedBinDir: string | undefined;

/** Set the managed bin directory so whichBin() checks it as fallback. */
export function setManagedBinDir(dir: string): void {
  managedBinDir = dir;
}

function whichBin(name: string): string | null {
  // 1. Check system PATH
  const found = Bun.which(name);
  if (found) return found;

  // 2. Check managed bin directory
  if (!managedBinDir) return null;

  const isWin = process.platform === "win32";
  const ext = isWin ? ".exe" : "";
  const candidates = [
    path.join(managedBinDir, name + ext),
    path.join(managedBinDir, "node_modules", ".bin", name + ext),
    ...(isWin
      ? [path.join(managedBinDir, "node_modules", ".bin", name + ".cmd")]
      : []),
  ];

  for (const candidate of candidates) {
    try {
      fsSync.accessSync(candidate);
      return candidate;
    } catch {
      // not found, try next
    }
  }
  return null;
}

// --- Server Configs ---

export const Deno: LspServerConfig = {
  id: "deno",
  extensions: [".ts", ".tsx", ".js", ".jsx", ".mjs"],
  root: async (file, projectRoot) => {
    // Deno only activates if deno.json exists
    for (const name of ["deno.json", "deno.jsonc"]) {
      const found = await findFileUp(name, path.dirname(file), projectRoot);
      if (found) return path.dirname(found);
    }
    return undefined;
  },
  async spawn(root) {
    const bin = whichBin("deno");
    if (!bin) return undefined;
    return {
      process: spawn(bin, ["lsp"], { cwd: root }),
    };
  },
};

export const Typescript: LspServerConfig = {
  id: "typescript",
  extensions: [".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".mts", ".cts"],
  install: { type: "npm", package: "typescript-language-server" },
  root: nearestRoot(
    [
      "package-lock.json",
      "bun.lockb",
      "bun.lock",
      "pnpm-lock.yaml",
      "yarn.lock",
    ],
    ["deno.json", "deno.jsonc"],
  ),
  async spawn(root) {
    const bin = whichBin("typescript-language-server");
    if (!bin) return undefined;
    return {
      process: spawn(bin, ["--stdio"], { cwd: root }),
    };
  },
};

export const Vue: LspServerConfig = {
  id: "vue",
  extensions: [".vue"],
  install: { type: "npm", package: "@vue/language-server" },
  root: nearestRoot([
    "package-lock.json",
    "bun.lockb",
    "bun.lock",
    "pnpm-lock.yaml",
    "yarn.lock",
  ]),
  async spawn(root) {
    const bin = whichBin("vue-language-server");
    if (!bin) return undefined;
    return {
      process: spawn(bin, ["--stdio"], { cwd: root }),
    };
  },
};

export const ESLint: LspServerConfig = {
  id: "eslint",
  extensions: [
    ".ts",
    ".tsx",
    ".js",
    ".jsx",
    ".mjs",
    ".cjs",
    ".mts",
    ".cts",
    ".vue",
  ],
  root: nearestRoot([
    "package-lock.json",
    "bun.lockb",
    "bun.lock",
    "pnpm-lock.yaml",
    "yarn.lock",
  ]),
  async spawn() {
    // ESLint LSP is a VS Code extension server, not a standalone binary.
    // Without auto-download it won't work for most users.
    // Included for forward-compatibility.
    return undefined;
  },
};

export const Oxlint: LspServerConfig = {
  id: "oxlint",
  extensions: [
    ".ts",
    ".tsx",
    ".js",
    ".jsx",
    ".mjs",
    ".cjs",
    ".mts",
    ".cts",
    ".vue",
    ".astro",
    ".svelte",
  ],
  root: nearestRoot([
    ".oxlintrc.json",
    "package-lock.json",
    "bun.lockb",
    "bun.lock",
    "pnpm-lock.yaml",
    "yarn.lock",
    "package.json",
  ]),
  async spawn(root) {
    let bin = whichBin("oxlint");
    if (bin) {
      return {
        process: spawn(bin, ["--lsp"], { cwd: root }),
      };
    }
    bin = whichBin("oxc_language_server");
    if (bin) {
      return {
        process: spawn(bin, [], { cwd: root }),
      };
    }
    return undefined;
  },
};

export const Biome: LspServerConfig = {
  id: "biome",
  extensions: [
    ".ts",
    ".tsx",
    ".js",
    ".jsx",
    ".mjs",
    ".cjs",
    ".mts",
    ".cts",
    ".json",
    ".jsonc",
    ".vue",
    ".astro",
    ".svelte",
    ".css",
    ".graphql",
    ".gql",
    ".html",
  ],
  root: nearestRoot([
    "biome.json",
    "biome.jsonc",
    "package-lock.json",
    "bun.lockb",
    "bun.lock",
    "pnpm-lock.yaml",
    "yarn.lock",
  ]),
  async spawn(root) {
    const bin = whichBin("biome");
    if (!bin) return undefined;
    return {
      process: spawn(bin, ["lsp-proxy", "--stdio"], { cwd: root }),
    };
  },
};

export const Gopls: LspServerConfig = {
  id: "gopls",
  extensions: [".go"],
  install: { type: "go", module: "golang.org/x/tools/gopls" },
  root: async (file, projectRoot) => {
    // go.work takes priority over go.mod
    const workRoot = await nearestRoot(["go.work"])(file, projectRoot);
    if (workRoot && workRoot !== projectRoot) return workRoot;
    // Check for go.work at project root level
    if (workRoot === projectRoot && (await fileExists(path.join(projectRoot, "go.work")))) {
      return workRoot;
    }
    return nearestRoot(["go.mod", "go.sum"])(file, projectRoot);
  },
  async spawn(root) {
    const bin = whichBin("gopls");
    if (!bin) return undefined;
    return {
      process: spawn(bin, [], { cwd: root }),
    };
  },
};

export const Rubocop: LspServerConfig = {
  id: "ruby-lsp",
  extensions: [".rb", ".rake", ".gemspec", ".ru"],
  install: { type: "gem", package: "rubocop" },
  root: nearestRoot(["Gemfile"]),
  async spawn(root) {
    const bin = whichBin("rubocop");
    if (!bin) return undefined;
    return {
      process: spawn(bin, ["--lsp"], { cwd: root }),
    };
  },
};

export const Ty: LspServerConfig = {
  id: "ty",
  extensions: [".py", ".pyi"],
  root: nearestRoot([
    "pyproject.toml",
    "ty.toml",
    "setup.py",
    "setup.cfg",
    "requirements.txt",
    "Pipfile",
    "pyrightconfig.json",
  ]),
  async spawn(root) {
    let binary = whichBin("ty");

    const initialization: Record<string, string> = {};
    const potentialVenvPaths = [
      process.env["VIRTUAL_ENV"],
      path.join(root, ".venv"),
      path.join(root, "venv"),
    ].filter((p): p is string => p !== undefined);

    for (const venvPath of potentialVenvPaths) {
      const pythonPath =
        process.platform === "win32"
          ? path.join(venvPath, "Scripts", "python.exe")
          : path.join(venvPath, "bin", "python");
      if (await fileExists(pythonPath)) {
        initialization["pythonPath"] = pythonPath;
        break;
      }
    }

    if (!binary) {
      for (const venvPath of potentialVenvPaths) {
        const tyPath =
          process.platform === "win32"
            ? path.join(venvPath, "Scripts", "ty.exe")
            : path.join(venvPath, "bin", "ty");
        if (await fileExists(tyPath)) {
          binary = tyPath;
          break;
        }
      }
    }

    if (!binary) return undefined;
    return {
      process: spawn(binary, ["server"], { cwd: root }),
      initialization,
    };
  },
};

export const Pyright: LspServerConfig = {
  id: "pyright",
  extensions: [".py", ".pyi"],
  install: { type: "npm", package: "pyright" },
  root: nearestRoot([
    "pyproject.toml",
    "setup.py",
    "setup.cfg",
    "requirements.txt",
    "Pipfile",
    "pyrightconfig.json",
  ]),
  async spawn(root) {
    const bin = whichBin("pyright-langserver");
    if (!bin) return undefined;

    const initialization: Record<string, string> = {};
    const potentialVenvPaths = [
      process.env["VIRTUAL_ENV"],
      path.join(root, ".venv"),
      path.join(root, "venv"),
    ].filter((p): p is string => p !== undefined);

    for (const venvPath of potentialVenvPaths) {
      const pythonPath =
        process.platform === "win32"
          ? path.join(venvPath, "Scripts", "python.exe")
          : path.join(venvPath, "bin", "python");
      if (await fileExists(pythonPath)) {
        initialization["pythonPath"] = pythonPath;
        break;
      }
    }

    return {
      process: spawn(bin, ["--stdio"], { cwd: root }),
      initialization,
    };
  },
};

export const ElixirLS: LspServerConfig = {
  id: "elixir-ls",
  extensions: [".ex", ".exs"],
  root: nearestRoot(["mix.exs", "mix.lock"]),
  async spawn(root) {
    const bin = whichBin("elixir-ls");
    if (!bin) {
      // Check for language_server.sh / .bat
      const script =
        process.platform === "win32"
          ? whichBin("language_server.bat")
          : whichBin("language_server.sh");
      if (!script) return undefined;
      return { process: spawn(script, [], { cwd: root }) };
    }
    return { process: spawn(bin, [], { cwd: root }) };
  },
};

export const Zls: LspServerConfig = {
  id: "zls",
  extensions: [".zig", ".zon"],
  install: {
    type: "github-release",
    repo: "zigtools/zls",
    assetSelector: (platform, arch) => {
      const cpu = arch === "arm64" ? "aarch64" : "x86_64";
      if (platform === "linux") return `${cpu}-linux`;
      if (platform === "darwin") return `${cpu}-macos`;
      if (platform === "win32") return `${cpu}-windows`;
      return undefined;
    },
    binaryName: "zls",
  },
  root: nearestRoot(["build.zig"]),
  async spawn(root) {
    const bin = whichBin("zls");
    if (!bin) return undefined;
    return { process: spawn(bin, [], { cwd: root }) };
  },
};

export const CSharp: LspServerConfig = {
  id: "csharp",
  extensions: [".cs"],
  install: { type: "dotnet", package: "csharp-ls" },
  root: nearestRoot([".sln", ".csproj", "global.json"]),
  async spawn(root) {
    const bin = whichBin("csharp-ls");
    if (!bin) return undefined;
    return { process: spawn(bin, [], { cwd: root }) };
  },
};

export const FSharp: LspServerConfig = {
  id: "fsharp",
  extensions: [".fs", ".fsi", ".fsx", ".fsscript"],
  install: { type: "dotnet", package: "fsautocomplete" },
  root: nearestRoot([".sln", ".fsproj", "global.json"]),
  async spawn(root) {
    const bin = whichBin("fsautocomplete");
    if (!bin) return undefined;
    return { process: spawn(bin, [], { cwd: root }) };
  },
};

export const SourceKit: LspServerConfig = {
  id: "sourcekit-lsp",
  extensions: [".swift", ".objc", ".objcpp"],
  root: nearestRoot(["Package.swift", "*.xcodeproj", "*.xcworkspace"]),
  async spawn(root) {
    const bin = whichBin("sourcekit-lsp");
    if (bin) {
      return { process: spawn(bin, [], { cwd: root }) };
    }
    // macOS: try xcrun
    if (process.platform !== "darwin") return undefined;
    const xcrun = whichBin("xcrun");
    if (!xcrun) return undefined;
    try {
      const { execSync } = await import("child_process");
      const lspPath = execSync("xcrun --find sourcekit-lsp", {
        encoding: "utf-8",
      }).trim();
      if (!lspPath) return undefined;
      return { process: spawn(lspPath, [], { cwd: root }) };
    } catch {
      return undefined;
    }
  },
};

export const RustAnalyzer: LspServerConfig = {
  id: "rust",
  extensions: [".rs"],
  install: {
    type: "github-release",
    repo: "rust-lang/rust-analyzer",
    assetSelector: (platform, arch) => {
      const cpu = arch === "arm64" ? "aarch64" : "x86_64";
      if (platform === "darwin") return `rust-analyzer-${cpu}-apple-darwin`;
      if (platform === "linux") return `rust-analyzer-${cpu}-unknown-linux-gnu`;
      if (platform === "win32") return `rust-analyzer-${cpu}-pc-windows-msvc`;
      return undefined;
    },
    binaryName: "rust-analyzer",
  },
  root: async (file, projectRoot) => {
    const crateRoot = await nearestRoot(["Cargo.toml", "Cargo.lock"])(
      file,
      projectRoot,
    );
    if (!crateRoot) return undefined;

    // Walk up to find [workspace] in Cargo.toml
    let currentDir = crateRoot;
    while (true) {
      const cargoPath = path.join(currentDir, "Cargo.toml");
      try {
        const content = await Bun.file(cargoPath).text();
        if (content.includes("[workspace]")) {
          return currentDir;
        }
      } catch {
        /* not found */
      }
      const parent = path.dirname(currentDir);
      if (parent === currentDir) break;
      if (!currentDir.startsWith(projectRoot)) break;
      currentDir = parent;
    }
    return crateRoot;
  },
  async spawn(root) {
    const bin = whichBin("rust-analyzer");
    if (!bin) return undefined;
    return { process: spawn(bin, [], { cwd: root }) };
  },
};

export const Clangd: LspServerConfig = {
  id: "clangd",
  extensions: [
    ".c",
    ".cpp",
    ".cc",
    ".cxx",
    ".c++",
    ".h",
    ".hpp",
    ".hh",
    ".hxx",
    ".h++",
  ],
  install: {
    type: "github-release",
    repo: "clangd/clangd",
    assetSelector: (platform, arch) => {
      if (platform === "linux") return "clangd-linux";
      if (platform === "darwin") return "clangd-mac";
      if (platform === "win32") return "clangd-windows";
      return undefined;
    },
    binaryName: "clangd",
  },
  root: nearestRoot([
    "compile_commands.json",
    "compile_flags.txt",
    ".clangd",
    "CMakeLists.txt",
    "Makefile",
  ]),
  async spawn(root) {
    const bin = whichBin("clangd");
    if (!bin) return undefined;
    return {
      process: spawn(bin, ["--background-index", "--clang-tidy"], {
        cwd: root,
      }),
    };
  },
};

export const Svelte: LspServerConfig = {
  id: "svelte",
  extensions: [".svelte"],
  install: { type: "npm", package: "svelte-language-server" },
  root: nearestRoot([
    "package-lock.json",
    "bun.lockb",
    "bun.lock",
    "pnpm-lock.yaml",
    "yarn.lock",
  ]),
  async spawn(root) {
    const bin = whichBin("svelteserver");
    if (!bin) return undefined;
    return { process: spawn(bin, ["--stdio"], { cwd: root }) };
  },
};

export const Astro: LspServerConfig = {
  id: "astro",
  extensions: [".astro"],
  install: { type: "npm", package: "@astrojs/language-server" },
  root: nearestRoot([
    "package-lock.json",
    "bun.lockb",
    "bun.lock",
    "pnpm-lock.yaml",
    "yarn.lock",
  ]),
  async spawn(root) {
    const bin = whichBin("astro-ls");
    if (!bin) return undefined;
    return { process: spawn(bin, ["--stdio"], { cwd: root }) };
  },
};

export const JDTLS: LspServerConfig = {
  id: "jdtls",
  extensions: [".java"],
  root: nearestRoot([
    "pom.xml",
    "build.gradle",
    "build.gradle.kts",
    ".project",
    ".classpath",
  ]),
  async spawn(root) {
    const java = whichBin("java");
    if (!java) return undefined;

    // Check for jdtls installation — user must install manually
    const jdtlsBin = whichBin("jdtls");
    if (jdtlsBin) {
      return { process: spawn(jdtlsBin, [], { cwd: root }) };
    }
    return undefined;
  },
};

export const KotlinLS: LspServerConfig = {
  id: "kotlin-ls",
  extensions: [".kt", ".kts"],
  root: async (file, projectRoot) => {
    const settingsRoot = await nearestRoot([
      "settings.gradle.kts",
      "settings.gradle",
    ])(file, projectRoot);
    if (settingsRoot && settingsRoot !== projectRoot) return settingsRoot;
    const wrapperRoot = await nearestRoot(["gradlew", "gradlew.bat"])(
      file,
      projectRoot,
    );
    if (wrapperRoot && wrapperRoot !== projectRoot) return wrapperRoot;
    const buildRoot = await nearestRoot([
      "build.gradle.kts",
      "build.gradle",
    ])(file, projectRoot);
    if (buildRoot) return buildRoot;
    return nearestRoot(["pom.xml"])(file, projectRoot);
  },
  async spawn(root) {
    const bin = whichBin("kotlin-lsp");
    if (!bin) return undefined;
    return { process: spawn(bin, ["--stdio"], { cwd: root }) };
  },
};

export const YamlLS: LspServerConfig = {
  id: "yaml-ls",
  extensions: [".yaml", ".yml"],
  install: { type: "npm", package: "yaml-language-server" },
  root: nearestRoot([
    "package-lock.json",
    "bun.lockb",
    "bun.lock",
    "pnpm-lock.yaml",
    "yarn.lock",
  ]),
  async spawn(root) {
    const bin = whichBin("yaml-language-server");
    if (!bin) return undefined;
    return { process: spawn(bin, ["--stdio"], { cwd: root }) };
  },
};

export const LuaLS: LspServerConfig = {
  id: "lua-ls",
  extensions: [".lua"],
  install: {
    type: "github-release",
    repo: "LuaLS/lua-language-server",
    assetSelector: (platform, arch) => {
      if (platform === "linux") return arch === "arm64" ? "linux-arm64" : "linux-x64";
      if (platform === "darwin") return arch === "arm64" ? "darwin-arm64" : "darwin-x64";
      if (platform === "win32") return "win32-x64";
      return undefined;
    },
    binaryName: "lua-language-server",
  },
  root: nearestRoot([
    ".luarc.json",
    ".luarc.jsonc",
    ".luacheckrc",
    ".stylua.toml",
    "stylua.toml",
    "selene.toml",
    "selene.yml",
  ]),
  async spawn(root) {
    const bin = whichBin("lua-language-server");
    if (!bin) return undefined;
    return { process: spawn(bin, [], { cwd: root }) };
  },
};

export const PHPIntelephense: LspServerConfig = {
  id: "php-intelephense",
  extensions: [".php"],
  install: { type: "npm", package: "intelephense" },
  root: nearestRoot(["composer.json", "composer.lock", ".php-version"]),
  async spawn(root) {
    const bin = whichBin("intelephense");
    if (!bin) return undefined;
    return {
      process: spawn(bin, ["--stdio"], { cwd: root }),
      initialization: {
        telemetry: { enabled: false },
      },
    };
  },
};

export const Prisma: LspServerConfig = {
  id: "prisma",
  extensions: [".prisma"],
  root: nearestRoot(["schema.prisma", "prisma/schema.prisma"]),
  async spawn(root) {
    const bin = whichBin("prisma");
    if (!bin) return undefined;
    return { process: spawn(bin, ["language-server"], { cwd: root }) };
  },
};

export const Dart: LspServerConfig = {
  id: "dart",
  extensions: [".dart"],
  root: nearestRoot(["pubspec.yaml", "analysis_options.yaml"]),
  async spawn(root) {
    const bin = whichBin("dart");
    if (!bin) return undefined;
    return {
      process: spawn(bin, ["language-server", "--lsp"], { cwd: root }),
    };
  },
};

export const Ocaml: LspServerConfig = {
  id: "ocaml-lsp",
  extensions: [".ml", ".mli"],
  root: nearestRoot(["dune-project", "dune-workspace", ".merlin", "opam"]),
  async spawn(root) {
    const bin = whichBin("ocamllsp");
    if (!bin) return undefined;
    return { process: spawn(bin, [], { cwd: root }) };
  },
};

export const BashLS: LspServerConfig = {
  id: "bash",
  extensions: [".sh", ".bash", ".zsh", ".ksh"],
  install: { type: "npm", package: "bash-language-server" },
  root: async (_file, projectRoot) => projectRoot,
  async spawn(root) {
    const bin = whichBin("bash-language-server");
    if (!bin) return undefined;
    return { process: spawn(bin, ["start"], { cwd: root }) };
  },
};

export const TerraformLS: LspServerConfig = {
  id: "terraform",
  extensions: [".tf", ".tfvars"],
  install: {
    type: "github-release",
    repo: "hashicorp/terraform-ls",
    assetSelector: (platform, arch) => {
      const cpu = arch === "arm64" ? "arm64" : "amd64";
      if (platform === "linux") return `terraform-ls_${cpu}`;
      if (platform === "darwin") return `terraform-ls_${cpu}`;
      if (platform === "win32") return `terraform-ls_${cpu}`;
      return undefined;
    },
    binaryName: "terraform-ls",
  },
  root: nearestRoot([
    ".terraform.lock.hcl",
    "terraform.tfstate",
    "*.tf",
  ]),
  async spawn(root) {
    const bin = whichBin("terraform-ls");
    if (!bin) return undefined;
    return {
      process: spawn(bin, ["serve"], { cwd: root }),
      initialization: {
        experimentalFeatures: {
          prefillRequiredFields: true,
          validateOnSave: true,
        },
      },
    };
  },
};

export const TexLab: LspServerConfig = {
  id: "texlab",
  extensions: [".tex", ".bib"],
  install: {
    type: "github-release",
    repo: "latex-lsp/texlab",
    assetSelector: (platform, arch) => {
      const cpu = arch === "arm64" ? "aarch64" : "x86_64";
      if (platform === "linux") return `texlab-${cpu}-linux`;
      if (platform === "darwin") return `texlab-${cpu}-macos`;
      if (platform === "win32") return `texlab-${cpu}-windows`;
      return undefined;
    },
    binaryName: "texlab",
  },
  root: nearestRoot([".latexmkrc", "latexmkrc", ".texlabroot", "texlabroot"]),
  async spawn(root) {
    const bin = whichBin("texlab");
    if (!bin) return undefined;
    return { process: spawn(bin, [], { cwd: root }) };
  },
};

export const DockerfileLS: LspServerConfig = {
  id: "dockerfile",
  extensions: [".dockerfile", "Dockerfile"],
  install: { type: "npm", package: "dockerfile-language-server-nodejs" },
  root: async (_file, projectRoot) => projectRoot,
  async spawn(root) {
    const bin = whichBin("docker-langserver");
    if (!bin) return undefined;
    return { process: spawn(bin, ["--stdio"], { cwd: root }) };
  },
};

export const GleamLS: LspServerConfig = {
  id: "gleam",
  extensions: [".gleam"],
  root: nearestRoot(["gleam.toml"]),
  async spawn(root) {
    const bin = whichBin("gleam");
    if (!bin) return undefined;
    return { process: spawn(bin, ["lsp"], { cwd: root }) };
  },
};

export const ClojureLSP: LspServerConfig = {
  id: "clojure-lsp",
  extensions: [".clj", ".cljs", ".cljc", ".edn"],
  root: nearestRoot([
    "deps.edn",
    "project.clj",
    "shadow-cljs.edn",
    "bb.edn",
    "build.boot",
  ]),
  async spawn(root) {
    let bin = whichBin("clojure-lsp");
    if (!bin && process.platform === "win32") {
      bin = whichBin("clojure-lsp.exe");
    }
    if (!bin) return undefined;
    return { process: spawn(bin, ["listen"], { cwd: root }) };
  },
};

export const Nixd: LspServerConfig = {
  id: "nixd",
  extensions: [".nix"],
  root: async (file, projectRoot) => {
    const flakeRoot = await nearestRoot(["flake.nix"])(file, projectRoot);
    if (flakeRoot && flakeRoot !== projectRoot) return flakeRoot;
    return projectRoot;
  },
  async spawn(root) {
    const bin = whichBin("nixd");
    if (!bin) return undefined;
    return { process: spawn(bin, [], { cwd: root }) };
  },
};

export const Tinymist: LspServerConfig = {
  id: "tinymist",
  extensions: [".typ", ".typc"],
  install: {
    type: "github-release",
    repo: "Myriad-Dreamin/tinymist",
    assetSelector: (platform, arch) => {
      const cpu = arch === "arm64" ? "aarch64" : "x86_64";
      if (platform === "linux") return `tinymist-${cpu}-unknown-linux-gnu`;
      if (platform === "darwin") return `tinymist-${cpu}-apple-darwin`;
      if (platform === "win32") return `tinymist-${cpu}-pc-windows-msvc`;
      return undefined;
    },
    binaryName: "tinymist",
  },
  root: nearestRoot(["typst.toml"]),
  async spawn(root) {
    const bin = whichBin("tinymist");
    if (!bin) return undefined;
    return { process: spawn(bin, [], { cwd: root }) };
  },
};

export const HLS: LspServerConfig = {
  id: "haskell-language-server",
  extensions: [".hs", ".lhs"],
  root: nearestRoot(["stack.yaml", "cabal.project", "hie.yaml", "*.cabal"]),
  async spawn(root) {
    const bin = whichBin("haskell-language-server-wrapper");
    if (!bin) return undefined;
    return { process: spawn(bin, ["--lsp"], { cwd: root }) };
  },
};

// All server configs — order matters for conflict resolution (Deno before Typescript)
export const ALL_SERVERS: LspServerConfig[] = [
  Deno,
  Typescript,
  Vue,
  ESLint,
  Oxlint,
  Biome,
  Gopls,
  Rubocop,
  Ty,
  Pyright,
  ElixirLS,
  Zls,
  CSharp,
  FSharp,
  SourceKit,
  RustAnalyzer,
  Clangd,
  Svelte,
  Astro,
  JDTLS,
  KotlinLS,
  YamlLS,
  LuaLS,
  PHPIntelephense,
  Prisma,
  Dart,
  Ocaml,
  BashLS,
  TerraformLS,
  TexLab,
  DockerfileLS,
  GleamLS,
  ClojureLSP,
  Nixd,
  Tinymist,
  HLS,
];
