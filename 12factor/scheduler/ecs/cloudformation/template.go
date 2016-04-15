package cloudformation

import (
	"bytes"
	"io"
	"text/template"

	"github.com/ghodss/yaml"
)

// YAMLTemplate takes a yaml string and returns a Template that will return json
// when executed.
func YAMLTemplate(t *template.Template) Template {
	return &yamlTemplate{
		Template: t,
	}
}

type yamlTemplate struct {
	*template.Template
}

func (t *yamlTemplate) Execute(w io.Writer, data interface{}) error {
	buf := new(bytes.Buffer)
	if err := t.Template.Execute(buf, data); err != nil {
		return err
	}

	raw, err := yaml.YAMLToJSON(buf.Bytes())
	if err != nil {
		return err
	}

	io.WriteString(w, string(raw))
	return nil
}
