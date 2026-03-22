package lsp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ServerConfig describes how to find and start an LSP server for a given language.
type ServerConfig struct {
	ID         string   // e.g., "go", "typescript"
	Extensions []string // e.g., [".go"], [".ts", ".tsx"]

	// InstallSpec describes how to auto-install the LSP binary. Nil means no auto-install.
	Install *InstallSpec

	// FindRoot locates the workspace root for this language by searching upward from filePath.
	FindRoot func(filePath, projectRoot string) (string, error)

	// SpawnCommand returns the command name and arguments to start the LSP server.
	SpawnCommand func(root string) (name string, args []string, err error)
}

// AllConfigs returns all registered server configurations.
func AllConfigs() []ServerConfig {
	return []ServerConfig{
		goConfig(),
		typescriptConfig(),
		pythonConfig(),
		rustConfig(),
		javaConfig(),
		cppConfig(),
		dartConfig(),
		rubyConfig(),
		phpConfig(),
		csharpConfig(),
	}
}

// ConfigForFile returns the best matching config for a file path, or nil if none matches.
func ConfigForFile(filePath string) *ServerConfig {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, cfg := range AllConfigs() {
		for _, e := range cfg.Extensions {
			if e == ext {
				c := cfg // copy
				return &c
			}
		}
	}
	return nil
}

func goConfig() ServerConfig {
	return ServerConfig{
		ID:         "go",
		Extensions: []string{".go"},
		Install:    &InstallSpec{Type: "go", Package: "golang.org/x/tools/gopls"},
		FindRoot: func(filePath, projectRoot string) (string, error) {
			return findFileUp("go.mod", filepath.Dir(filePath), projectRoot)
		},
		SpawnCommand: func(root string) (string, []string, error) {
			bin := whichBin("gopls")
			if bin == "" {
				return "", nil, fmt.Errorf("gopls not found in PATH")
			}
			return bin, []string{"serve"}, nil
		},
	}
}

func typescriptConfig() ServerConfig {
	return ServerConfig{
		ID:         "typescript",
		Extensions: []string{".ts", ".tsx", ".js", ".jsx"},
		Install:    &InstallSpec{Type: "npm", Package: "typescript-language-server"},
		FindRoot: func(filePath, projectRoot string) (string, error) {
			root, err := findFileUp("tsconfig.json", filepath.Dir(filePath), projectRoot)
			if err == nil {
				return root, nil
			}
			return findFileUp("package.json", filepath.Dir(filePath), projectRoot)
		},
		SpawnCommand: func(root string) (string, []string, error) {
			bin := whichBin("typescript-language-server")
			if bin == "" {
				return "", nil, fmt.Errorf("typescript-language-server not found in PATH")
			}
			return bin, []string{"--stdio"}, nil
		},
	}
}

func pythonConfig() ServerConfig {
	return ServerConfig{
		ID:         "python",
		Extensions: []string{".py"},
		Install:    &InstallSpec{Type: "npm", Package: "pyright", Binary: "pyright-langserver"},
		FindRoot: func(filePath, projectRoot string) (string, error) {
			for _, marker := range []string{"pyproject.toml", "setup.py", "requirements.txt"} {
				root, err := findFileUp(marker, filepath.Dir(filePath), projectRoot)
				if err == nil {
					return root, nil
				}
			}
			return projectRoot, nil
		},
		SpawnCommand: func(root string) (string, []string, error) {
			if bin := whichBin("pyright-langserver"); bin != "" {
				return bin, []string{"--stdio"}, nil
			}
			if bin := whichBin("pylsp"); bin != "" {
				return bin, nil, nil
			}
			return "", nil, fmt.Errorf("pyright-langserver or pylsp not found in PATH")
		},
	}
}

func rustConfig() ServerConfig {
	return ServerConfig{
		ID:         "rust",
		Extensions: []string{".rs"},
		Install:    &InstallSpec{Type: "github-release", Package: "rust-lang/rust-analyzer", Binary: "rust-analyzer"},
		FindRoot: func(filePath, projectRoot string) (string, error) {
			return findFileUp("Cargo.toml", filepath.Dir(filePath), projectRoot)
		},
		SpawnCommand: func(root string) (string, []string, error) {
			bin := whichBin("rust-analyzer")
			if bin == "" {
				return "", nil, fmt.Errorf("rust-analyzer not found in PATH")
			}
			return bin, nil, nil
		},
	}
}

func javaConfig() ServerConfig {
	return ServerConfig{
		ID:         "java",
		Extensions: []string{".java"},
		FindRoot: func(filePath, projectRoot string) (string, error) {
			for _, marker := range []string{"pom.xml", "build.gradle", "build.gradle.kts"} {
				root, err := findFileUp(marker, filepath.Dir(filePath), projectRoot)
				if err == nil {
					return root, nil
				}
			}
			return projectRoot, nil
		},
		SpawnCommand: func(root string) (string, []string, error) {
			bin := whichBin("jdtls")
			if bin == "" {
				return "", nil, fmt.Errorf("jdtls not found in PATH")
			}
			return bin, nil, nil
		},
	}
}

func cppConfig() ServerConfig {
	return ServerConfig{
		ID:         "cpp",
		Extensions: []string{".c", ".cpp", ".cc", ".cxx", ".h", ".hpp"},
		Install:    &InstallSpec{Type: "github-release", Package: "clangd/clangd", Binary: "clangd"},
		FindRoot: func(filePath, projectRoot string) (string, error) {
			for _, marker := range []string{"compile_commands.json", "CMakeLists.txt"} {
				root, err := findFileUp(marker, filepath.Dir(filePath), projectRoot)
				if err == nil {
					return root, nil
				}
			}
			return projectRoot, nil
		},
		SpawnCommand: func(root string) (string, []string, error) {
			bin := whichBin("clangd")
			if bin == "" {
				return "", nil, fmt.Errorf("clangd not found in PATH")
			}
			return bin, nil, nil
		},
	}
}

func dartConfig() ServerConfig {
	return ServerConfig{
		ID:         "dart",
		Extensions: []string{".dart"},
		FindRoot: func(filePath, projectRoot string) (string, error) {
			return findFileUp("pubspec.yaml", filepath.Dir(filePath), projectRoot)
		},
		SpawnCommand: func(root string) (string, []string, error) {
			bin := whichBin("dart")
			if bin == "" {
				return "", nil, fmt.Errorf("dart not found in PATH")
			}
			return bin, []string{"language-server", "--protocol=lsp"}, nil
		},
	}
}

func rubyConfig() ServerConfig {
	return ServerConfig{
		ID:         "ruby",
		Extensions: []string{".rb"},
		FindRoot: func(filePath, projectRoot string) (string, error) {
			root, err := findFileUp("Gemfile", filepath.Dir(filePath), projectRoot)
			if err != nil {
				return projectRoot, nil
			}
			return root, nil
		},
		SpawnCommand: func(root string) (string, []string, error) {
			bin := whichBin("solargraph")
			if bin == "" {
				return "", nil, fmt.Errorf("solargraph not found in PATH")
			}
			return bin, []string{"stdio"}, nil
		},
	}
}

func phpConfig() ServerConfig {
	return ServerConfig{
		ID:         "php",
		Extensions: []string{".php"},
		FindRoot: func(filePath, projectRoot string) (string, error) {
			root, err := findFileUp("composer.json", filepath.Dir(filePath), projectRoot)
			if err != nil {
				return projectRoot, nil
			}
			return root, nil
		},
		SpawnCommand: func(root string) (string, []string, error) {
			bin := whichBin("intelephense")
			if bin == "" {
				return "", nil, fmt.Errorf("intelephense not found in PATH")
			}
			return bin, []string{"--stdio"}, nil
		},
	}
}

func csharpConfig() ServerConfig {
	return ServerConfig{
		ID:         "csharp",
		Extensions: []string{".cs"},
		FindRoot: func(filePath, projectRoot string) (string, error) {
			for _, marker := range []string{"*.sln", "*.csproj"} {
				root, err := findGlobUp(marker, filepath.Dir(filePath), projectRoot)
				if err == nil {
					return root, nil
				}
			}
			return projectRoot, nil
		},
		SpawnCommand: func(root string) (string, []string, error) {
			bin := whichBin("OmniSharp")
			if bin == "" {
				return "", nil, fmt.Errorf("OmniSharp not found in PATH")
			}
			return bin, []string{"-lsp"}, nil
		},
	}
}

// findFileUp searches for a file named `name` starting from `start` directory,
// going up until `stop` directory (inclusive). Returns the directory containing the file.
func findFileUp(name, start, stop string) (string, error) {
	start, _ = filepath.Abs(start)
	stop, _ = filepath.Abs(stop)

	dir := start
	for {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}

		if strings.EqualFold(filepath.Clean(dir), filepath.Clean(stop)) {
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("file %q not found from %s to %s", name, start, stop)
}

// findGlobUp searches for files matching a glob pattern starting from `start` directory,
// going up until `stop` directory (inclusive).
func findGlobUp(pattern, start, stop string) (string, error) {
	start, _ = filepath.Abs(start)
	stop, _ = filepath.Abs(stop)

	dir := start
	for {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err == nil && len(matches) > 0 {
			return dir, nil
		}

		if strings.EqualFold(filepath.Clean(dir), filepath.Clean(stop)) {
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("pattern %q not found from %s to %s", pattern, start, stop)
}

// whichBin finds a binary in PATH or in the managed bin directory.
// Returns empty string if not found.
func whichBin(name string) string {
	// First try PATH
	if p := lookPath(name); p != "" {
		return p
	}

	// Then try managed bin dir (direct path)
	binDir := ManagedBinDir()
	if p := lookInDir(binDir, name); p != "" {
		return p
	}

	// Then try managed bin dir node_modules/.bin/ (for npm-installed tools)
	nmBin := filepath.Join(binDir, "node_modules", ".bin")
	if p := lookInDir(nmBin, name); p != "" {
		return p
	}

	return ""
}

// lookPath finds a binary using exec.LookPath with Windows extension handling.
func lookPath(name string) string {
	if runtime.GOOS == "windows" {
		for _, ext := range []string{"", ".exe", ".cmd", ".bat"} {
			path, err := exec.LookPath(name + ext)
			if err == nil {
				return path
			}
		}
		return ""
	}

	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

// lookInDir checks if a binary exists in a specific directory.
func lookInDir(dir, name string) string {
	if runtime.GOOS == "windows" {
		for _, ext := range []string{".exe", ".cmd", ".bat", ""} {
			candidate := filepath.Join(dir, name+ext)
			if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
				return candidate
			}
		}
		return ""
	}

	candidate := filepath.Join(dir, name)
	fi, err := os.Stat(candidate)
	if err != nil {
		return ""
	}
	if fi.IsDir() {
		return ""
	}
	return candidate
}
