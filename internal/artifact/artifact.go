package artifact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const BaseDir = "artifacts"

func Write(subdir, filename, content string) error {
	dir := filepath.Join(BaseDir, subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	path := filepath.Join(dir, filename)
	return os.WriteFile(path, []byte(content), 0o644)
}

func Read(subdir, filename string) (string, error) {
	path := filepath.Join(BaseDir, subdir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(data), nil
}

func ReadAll(subdir string) (map[string]string, error) {
	dir := filepath.Join(BaseDir, subdir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("readdir %s: %w", dir, err)
	}

	result := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		result[e.Name()] = string(data)
	}
	return result, nil
}

func ReadDir(subdir string) (string, error) {
	files, err := ReadAll(subdir)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for name, content := range files {
		fmt.Fprintf(&b, "=== %s ===\n%s\n\n", name, content)
	}
	return b.String(), nil
}
