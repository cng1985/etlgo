package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolveUnder(dataDir, p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("路径不能为空")
	}
	if filepath.IsAbs(p) {
		return "", fmt.Errorf("只允许使用数据目录内的相对路径")
	}
	cleaned := filepath.Clean(p)
	if cleaned == "." || strings.HasPrefix(cleaned, "..") {
		return "", fmt.Errorf("路径越界")
	}
	root, err := filepath.Abs(dataDir)
	if err != nil {
		return "", err
	}
	full := filepath.Join(root, cleaned)
	rel, err := filepath.Rel(root, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("路径越界")
	}
	return full, nil
}

func EnsureParent(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}
