package mfmap_test

import (
	"reflect"
	"testing"

	"gometeo/mfmap"
	"gometeo/testutils"
)

func TestMapParser(t *testing.T) {
	var mapParseTests = map[string]struct {
		want interface{}
		got  func(j *mfmap.MapData) interface{}
	}{
		"Info.Taxonomy": {
			want: "PAYS",
			got:  func(j *mfmap.MapData) interface{} { return j.Info.Taxonomy },
		},
		"Info.IdTechnique": {
			want: "PAYS007",
			got:  func(j *mfmap.MapData) interface{} { return j.Info.IdTechnique },
		},
		"Tools.Config.Site": {
			want: "rwg",
			got:  func(j *mfmap.MapData) interface{} { return j.Tools.Config.Site },
		},
		"Tools.Config.BaseUrl": {
			want: "meteofrance.com/internet2018client/2.0",
			got:  func(j *mfmap.MapData) interface{} { return j.Tools.Config.BaseUrl },
		},
		"ChildrenPOI": {
			want: "VILLE_FRANCE",
			got:  func(j *mfmap.MapData) interface{} { return j.Children[0].Taxonomy },
		},
		"Subzone": {
			want: mfmap.Subzone{
				Path: "/previsions-meteo-france/auvergne-rhone-alpes/10",
				Name: "Auvergne-Rhône-Alpes",
			},
			got: func(j *mfmap.MapData) interface{} { return j.Subzones["REGIN10"] },
		},
	}
	data := testParseMap(t)
	for key, test := range mapParseTests {
		t.Run(key, func(t *testing.T) {
			got := test.got(data)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("%s got '%s' want '%s'", key, got, test.want)
			}
		})
	}
}

func TestMapParserFail(t *testing.T) {
	_, err := mfmap.ParseData(testutils.JsonFailReader(t))
	if err == nil {
		t.Error("error expected")
		return
	}
}

func testParseMap(t *testing.T) *mfmap.MapData {
	m := mfmap.MfMap{Conf: testutils.TestConf}
	if err := m.ParseHtml(testutils.HtmlReader(t)); err != nil {
		t.Fatalf("ParseHtml() error: %s", err)
	}
	return m.Data
}

func TestName(t *testing.T) {
	m := mfmap.MfMap{
		Data: testParseMap(t),
	}
	t.Run("name", func(t *testing.T) {
		want := "France"
		got := m.Name()
		if got != want {
			t.Fatalf("MfMap.Name() got '%s' want '%s'", got, want)
		}
	})
	t.Run("path", func(t *testing.T) {
		want := "france"
		got := m.Path()
		if got != want {
			t.Fatalf("MfMap.Path() got '%s' want '%s'", got, want)
		}
	})
}
