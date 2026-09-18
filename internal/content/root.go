package content

import (
	"fmt"
	"os"
	"path/filepath"
)

// ManifestName 是题库根清单文件名。
const ManifestName = "clablings.yaml"

// FindRoot 从 start 目录向上查找含 clablings.yaml 的目录。
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
			return "", fmt.Errorf("未找到题库根：从 %s 向上未找到 %s", start, ManifestName)
		}
		dir = parent
	}
}
