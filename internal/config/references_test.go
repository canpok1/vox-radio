package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/canpok1/vox-radio/internal/config"
)

// refDocAllowlist lists yaml field names that are intentionally undocumented in a
// given reference file (e.g. internal-only fields with no user-facing meaning).
// Add entries here with a reason comment instead of silently skipping the check.
var refDocAllowlist = map[string]map[string]string{
	"vox-radio.md":    {},
	"episode-spec.md": {},
	"assets.md":       {},
}

// collectYAMLFieldNames walks t (following pointers, slices, arrays, and map values)
// and collects every yaml tag name found on struct fields. visited guards against
// infinite recursion on self-referential types (e.g. EpisodeCondition.Not).
func collectYAMLFieldNames(t reflect.Type, names map[string]bool, visited map[reflect.Type]bool) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:
		if visited[t] {
			return
		}
		visited[t] = true
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag, ok := f.Tag.Lookup("yaml")
			if !ok {
				continue
			}
			name, _, _ := strings.Cut(tag, ",")
			if name == "" || name == "-" {
				continue
			}
			names[name] = true
			collectYAMLFieldNames(f.Type, names, visited)
		}
	case reflect.Slice, reflect.Array, reflect.Map:
		collectYAMLFieldNames(t.Elem(), names, visited)
	}
}

// TestConfigFieldsDocumentedInReferences は internal/config の設定構造体が持つ全 yaml
// フィールド名を、対応する references/*.md にバックティック表記（例: `field_name`）で
// 記載されているかを突き合わせる。未記載フィールドがあれば、references の更新漏れ
// （PR #548 のような事象）として検知する。
//
// 意図的にドキュメント化しないフィールドがある場合は、削除するのではなく
// refDocAllowlist に理由付きで追加すること。
func TestConfigFieldsDocumentedInReferences(t *testing.T) {
	roots := []struct {
		typ     reflect.Type
		docFile string
	}{
		{reflect.TypeOf(config.Config{}), "vox-radio.md"},
		{reflect.TypeOf(config.EpisodeSpec{}), "episode-spec.md"},
		{reflect.TypeOf(config.AssetsConfig{}), "assets.md"},
	}

	for _, root := range roots {
		t.Run(root.docFile, func(t *testing.T) {
			fields := map[string]bool{}
			collectYAMLFieldNames(root.typ, fields, map[reflect.Type]bool{})

			docPath := filepath.Join("..", "cli", "skills", "vox-radio", "references", root.docFile)
			content, err := os.ReadFile(docPath)
			if err != nil {
				t.Fatalf("reading %s: %v", docPath, err)
			}
			doc := string(content)
			allowed := refDocAllowlist[root.docFile]

			names := make([]string, 0, len(fields))
			for name := range fields {
				names = append(names, name)
			}
			sort.Strings(names)

			var missing []string
			for _, name := range names {
				if _, ok := allowed[name]; ok {
					continue
				}
				if !strings.Contains(doc, "`"+name+"`") {
					missing = append(missing, name)
				}
			}
			if len(missing) > 0 {
				t.Errorf("%s に未記載のフィールドがあります（表に `フィールド名` の形式で追記するか、意図的なら refDocAllowlist に理由付きで追加してください）: %s",
					docPath, strings.Join(missing, ", "))
			}
		})
	}
}
