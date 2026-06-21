package gather

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/canpok1/vox-radio/internal/model"
)

// fetchText reads a text file and returns it as a single Article with Body set to the file content.
// title is used as the article title; when empty the filename without extension is used.
func fetchText(filePath, title string) (*model.Article, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read text file %q: %w", filePath, err)
	}
	body := string(data)
	if title == "" {
		base := filepath.Base(filePath)
		title = strings.TrimSuffix(base, filepath.Ext(base))
	}
	return &model.Article{
		DedupKey: TextDedupKey(filePath, title, body),
		Title:    title,
		Body:     body,
	}, nil
}
