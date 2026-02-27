export interface NpmInstallSpec {
  type: "npm";
  /** npm package name, e.g. "typescript-language-server" */
  package: string;
}

export interface GoInstallSpec {
  type: "go";
  /** Full Go module path, e.g. "golang.org/x/tools/gopls" */
  module: string;
  /** Version tag, default "@latest" */
  version?: string;
}

export interface GithubReleaseSpec {
  type: "github-release";
  /** GitHub repo, e.g. "rust-lang/rust-analyzer" */
  repo: string;
  /**
   * Select the correct asset name from a release.
   * Returns asset filename substring to match, or undefined if platform unsupported.
   */
  assetSelector: (
    platform: NodeJS.Platform,
    arch: string,
  ) => string | undefined;
  /** Binary name inside the archive (if different from server binary name) */
  binaryName?: string;
}

export interface GemInstallSpec {
  type: "gem";
  /** Gem package name, e.g. "rubocop" */
  package: string;
}

export interface DotnetToolSpec {
  type: "dotnet";
  /** Dotnet tool package name, e.g. "csharp-ls" */
  package: string;
}

export type InstallSpec =
  | NpmInstallSpec
  | GoInstallSpec
  | GithubReleaseSpec
  | GemInstallSpec
  | DotnetToolSpec;

export interface InstallResult {
  success: boolean;
  /** Path to the installed binary, if successful */
  binaryPath?: string;
  /** Error message if failed */
  error?: string;
}
