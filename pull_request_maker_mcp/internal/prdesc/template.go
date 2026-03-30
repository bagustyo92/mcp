package prdesc

import (
	"embed"
	"strings"
)

//go:embed templates/*.md
var embeddedTemplates embed.FS

// TemplateLoader loads PR description templates from embedded files or custom paths.
type TemplateLoader struct {
	templates map[string]string
}

// NewTemplateLoader creates a TemplateLoader with built-in templates.
func NewTemplateLoader() *TemplateLoader {
	loader := &TemplateLoader{
		templates: make(map[string]string),
	}

	// Load embedded templates
	for _, name := range []string{"comprehensive", "concise"} {
		data, err := embeddedTemplates.ReadFile("templates/" + name + ".md")
		if err == nil {
			loader.templates[name] = string(data)
		}
	}

	return loader
}

// Load returns the template for the given mode.
// Falls back to "comprehensive" if the mode is unknown.
func (l *TemplateLoader) Load(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))

	if tmpl, ok := l.templates[mode]; ok {
		return tmpl
	}

	// Default fallback
	if tmpl, ok := l.templates["comprehensive"]; ok {
		return tmpl
	}

	return "## Summary\n\nDescribe the changes made in this PR.\n"
}
