package formatter

import (
	"bytes"
	"text/template"
)

type TemplateFormatter struct {
	tmpl *template.Template
}

func NewTemplateFormatter(templateText string) (*TemplateFormatter, error) {
	t, err := template.New("secret").Parse(templateText)
	if err != nil {
		return nil, err
	}

	return &TemplateFormatter{
		tmpl: t,
	}, nil
}

func (t *TemplateFormatter) Format(data map[string]any) ([]byte, error) {
	var result bytes.Buffer
	if err := t.tmpl.Execute(&result, data); err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
