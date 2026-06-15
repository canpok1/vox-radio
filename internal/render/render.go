package render

import (
	"bytes"
	"text/template"

	"github.com/canpok1/vox-radio/internal/model"
)

// Render executes the given text/template source with the manifest as data context.
// FuncMap provides:
//   - corner(id string) *model.ManifestCorner — returns the corner with the given ID (nil if not found)
//   - hasLinks(c model.ManifestCorner) bool — returns true if any article has a non-empty URL
func Render(tmplText string, m model.Manifest) (string, error) {
	funcMap := template.FuncMap{
		"corner": func(id string) *model.ManifestCorner {
			for i := range m.Corners {
				if m.Corners[i].ID == id {
					return &m.Corners[i]
				}
			}
			return nil
		},
		"hasLinks": func(c model.ManifestCorner) bool {
			for _, a := range c.Articles {
				if a.URL != "" {
					return true
				}
			}
			return false
		},
	}

	t, err := template.New("").Funcs(funcMap).Parse(tmplText)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, m); err != nil {
		return "", err
	}
	return buf.String(), nil
}
