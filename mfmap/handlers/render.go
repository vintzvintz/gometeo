package handlers

import (
	_ "embed"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	gj "gometeo/geojson"
	"gometeo/mfmap"
)

// TemplateData contains data for htmlTemplate.Execute()
type TemplateData struct {
	Description string
	Title       string
	Path        string
	VueJs       string
	CacheId     string
	Message     string
}

// messageFile is the path to the optional message file.
// Exported for testing.
var messageFile = "message.txt"

// readMessage returns the content of messageFile, or "" if the file
// is missing, empty, or contains only whitespace.
func readMessage() string {
	b, err := os.ReadFile(messageFile)
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(b))
	return s
}

//go:embed template.html
var templateFile string

// htmlTemplate is a global html/template for html rendering
var htmlTemplate = template.Must(template.New("").Parse(templateFile))

// WriteHtml renders the HTML page for m into wr.
func WriteHtml(wr io.Writer, m *mfmap.MfMap) error {
	return htmlTemplate.Execute(wr, &TemplateData{
		Description: fmt.Sprintf("Météo pour la zone %s sur une page grande et unique", m.Data.Info.Name),
		Title:       fmt.Sprintf("Météo %s", m.Data.Info.Name),
		Path:        m.Path(),
		CacheId:     m.Conf.CacheId,
		VueJs:       m.Conf.VueJs,
		Message:     readMessage(),
	})
}

type jsonMap struct {
	Name       string         `json:"name"`
	Path       string         `json:"path"`
	Breadcrumb mfmap.Breadcrumbs `json:"breadcrumb"`
	Idtech     string         `json:"idtech"`
	Taxonomy   string         `json:"taxonomy"`
	Bbox       gj.Bbox        `json:"bbox"`
	SubZones   gj.GeoFeatures `json:"subzones"`
	Prevs      gj.PrevList    `json:"prevs"`
	Chroniques gj.Graphdata   `json:"chroniques"`
}

// WriteJson writes all forecast data available in m as a JSON object into wr.
func WriteJson(wr io.Writer, m *mfmap.MfMap) error {
	obj, err := BuildJson(m)
	if err != nil {
		return err
	}
	b, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = io.Copy(wr, bytes.NewReader(b))
	return err
}

// BuildJson builds the JSON response object for m.
func BuildJson(m *mfmap.MfMap) (*jsonMap, error) {
	cr := mfmap.CropRatio
	bbox := m.Geography.Bbox.Crop(cr.Left, cr.Right, cr.Top, cr.Bottom)

	j := jsonMap{
		Name:       m.Name(),
		Path:       m.Path(),
		Breadcrumb: m.Breadcrumb,
		Idtech:     m.Data.Info.IdTechnique,
		Taxonomy:   m.Data.Info.Taxonomy,
		SubZones:   m.Geography.Features,
		Bbox:       bbox,
		Prevs:      m.Prevs,
		Chroniques: m.Graphdata,
	}
	// highchart disabled for PAYS. Only on DEPTs & REGIONs
	if m.Data.Info.Taxonomy == "PAYS" {
		j.Chroniques = nil
	}
	return &j, nil
}
