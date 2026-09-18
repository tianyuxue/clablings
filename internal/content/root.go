package content

import (
	"fmt"
	"os"
	"path/filepath"
)

// ManifestName is the library root manifest file name.
const ManifestName = "clablings.yaml"

// FindRoot searches upward from the start directory for a directory containing clablings.yaml.
func FindRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if st, err := os.Stat(filepath.Join(dir, ManifestName)); err == nil && !st.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("library root not found: no %s found searching upward from %s", ManifestName, start)
		}
		dir = parent
	}
}
