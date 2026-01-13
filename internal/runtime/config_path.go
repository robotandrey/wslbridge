package runtime

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	LocalConfigDir  = ".values"
	LocalConfigFile = "values.local.yaml"
)

// FindProjectRoot walks up from startDir looking for .git or go.mod.
// If nothing found, returns startDir.
func FindProjectRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		if exists(filepath.Join(dir, ".git")) || exists(filepath.Join(dir, "go.mod")) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return dir, nil
		}
		dir = parent
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ResolveProjectLocalConfigPath returns <projectRoot>/.values/values.local.yaml
func ResolveProjectLocalConfigPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, err := FindProjectRoot(cwd)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, LocalConfigDir, LocalConfigFile), nil
}

// ResolveConfigPath prefers project-local config when it looks like a wslbridge repo,
// otherwise falls back to the user config path.
func ResolveConfigPath() (string, error) {
	paths, err := DefaultPaths()
	if err != nil {
		return "", err
	}
	localPath, err := ResolveProjectLocalConfigPath()
	if err != nil {
		return "", err
	}
	if shouldUseLocalConfig(localPath) {
		return localPath, nil
	}
	return paths.ConfigPath, nil
}

func shouldUseLocalConfig(localPath string) bool {
	if exists(localPath) {
		return true
	}
	root := filepath.Dir(filepath.Dir(localPath))
	return isWslbridgeRepo(root)
}

func isWslbridgeRepo(root string) bool {
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return false
	}
	return parseModulePath(string(b)) == "wslbridge"
}

func parseModulePath(goMod string) string {
	for _, line := range strings.Split(goMod, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}
