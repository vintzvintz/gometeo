package mfmap

import (
	_ "embed"
	"fmt"
	"gometeo/appconf"
	"io"
	"text/template"
)

//go:embed template.html
var templateFile string

// TemplateData contains data for htmlTemplate.Execute()
type TemplateData struct {
	Description string
	Title       string
	Path        string
	VueJs       string
	CacheId     string
}

// htmlTemplate is a global html/template for html rendering
// this global variable is set up once at startup by the init() function
var htmlTemplate = template.Must(template.New("").Parse(templateFile))

// main html file
func (m *MfMap) WriteHtml(wr io.Writer) error {

	title := fmt.Sprintf("Météo %s", m.Data.Info.Name)
	desc := fmt.Sprintf("Météo pour la zone %s sur une page grande et unique", m.Data.Info.Name)
	path := m.Path()
	vue := "vue.esm-browser.dev.js"
	if appconf.VueProd() {
		vue = "vue.esm-browser.prod.js"
	}

	return htmlTemplate.Execute(wr, &TemplateData{
		Description: desc,
		Title:       title,
		Path:        path,
		CacheId:     appconf.CacheId(),
		VueJs:       vue,
	})
}
