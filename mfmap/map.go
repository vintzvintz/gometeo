package mfmap

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// accessor
func (m *MfMap) SetParent(parent *MfMap) {
	m.parent = parent
}

func NewFrom(r io.Reader) (*MfMap, error) {
	j, err := JsonFilter(r)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(j)
	if err != nil {
		return nil, err
	}
	return &MfMap{data: data}, nil
}

// isJsonTag detecte l'élement contenant les donnnées drupal
// <script type="application/json" data-drupal-selector="drupal-settings-json">
func isJsonTag(t html.Token) bool {
	isScript := (t.Type == html.StartTagToken) && (t.Data == "script")
	if !isScript {
		return false
	}
	var hasAttrType, hasDrupalAttr bool
	for _, a := range t.Attr {
		switch a.Key {
		case "type":
			hasAttrType = (a.Val == "application/json")
		case "data-drupal-selector":
			hasDrupalAttr = (a.Val == "drupal-settings-json")
		}
	}
	return isScript && hasAttrType && hasDrupalAttr
}

// JsonFilter extracts json data from an html page
func JsonFilter(src io.Reader) (io.Reader, error) {
	z := html.NewTokenizer(src)
	var inJson bool
loop:
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken: // éventuellement io.EOF
			break loop
		case html.StartTagToken:
			inJson = isJsonTag(z.Token())
		case html.TextToken:
			if inJson {
				return strings.NewReader(z.Token().Data), nil
			}
		}
	}
	return nil, fmt.Errorf("données JSON non trouvées")
}

func JsonParser(r io.Reader) (*JsonData, error) {
	var j JsonData
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(buf, &j)
	return &j, err
}

// ApiURL builds API URL from "config" node
// typically : https://rpcache-aa.meteofrance.com/internet2018client/2.0
func (j *JsonData) ApiURL() string {
	conf := j.ToolsCommon.Config
	return fmt.Sprintf("https://%s.%s", conf.Site, conf.BaseUrl)
}
