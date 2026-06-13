package config

import (
	"fmt"
	"net/url"
	"path/filepath"
)

// resolveFileURL normalizes a file:// URL to an absolute file:// URL.
// Relative paths within file:// URLs are resolved relative to specDir.
// Non-file scheme URLs are returned as-is.
func resolveFileURL(specDir, raw string) (string, error) {
	if raw == "" {
		return raw, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse URL %q: %w", raw, err)
	}
	if u.Scheme != "file" {
		return raw, nil
	}

	// For file://host/path, u.Host + u.Path reconstructs the local path.
	// For file:///abs/path, u.Host is empty and u.Path is the absolute path.
	localPath := u.Host + u.Path

	resolved := resolveFile(specDir, localPath)

	absPath, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve file URL %q: %w", raw, err)
	}

	return "file://" + absPath, nil
}
