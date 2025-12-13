package runtime

import (
	"os"
	"path/filepath"
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

// ResolveProjectLocalConfigPath returns <projectRoot>/values/values.local.yaml
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
