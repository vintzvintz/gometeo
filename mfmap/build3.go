package mfmap

import (
	_ "embed"
	"fmt"
	"io"
	"text/template"
)

//go:embed template.html
var templateFile string

// TemplateData contains data for htmlTemplate.Execute()
type TemplateData struct {
	HeadDescription string
	HeadTitle       string
	Path            string
	VueJs           string
}

// htmlTemplate is a global html/template for html rendering
// this global variable is set up once at startup by the init() function
var htmlTemplate = template.Must(template.New("").Parse(templateFile))

// main html file
func (m *MfMap) WriteHtml(wr io.Writer) error {
	return htmlTemplate.Execute(wr, &TemplateData{
		HeadDescription: fmt.Sprintf("Météo pour la zone %s sur une page grande et unique", m.Data.Info.Name),
		HeadTitle:       fmt.Sprintf("Météo %s", m.Data.Info.Name),
		Path:            m.Path(),
		VueJs:           "vue.esm-browser.js",
	})
}
