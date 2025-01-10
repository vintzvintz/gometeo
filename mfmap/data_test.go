package mfmap

import (
	"encoding/json"
	"gometeo/testutils"
	"reflect"
	"testing"
)

func TestStringFloat(t *testing.T) {
	type Item struct {
		A stringFloat `json:"a"`
		B stringFloat `json:"b"`
		C stringFloat `json:"c"`
		D stringFloat `json:"d"`
	}
	jsonData := []byte(`{"a":null, "c":"51", "d":51}`) // b is missing
	want := Item{A: 0, B: 0, C: 51, D: 51}
	var item Item
	err := json.Unmarshal(jsonData, &item)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(item, want) {
		t.Errorf("stringFloat custom Unmarshall() got %v, want %v", item, want)
	}
}

func TestMapParser(t *testing.T) {
	var mapParseTests = map[string]struct {
		want interface{}
		got  func(j *MapData) interface{}
	}{
		"Info.Taxonomy": {
			want: "PAYS",
			got:  func(j *MapData) interface{} { return j.Info.Taxonomy },
		},
		"Info.IdTechnique": {
			want: "PAYS007",
			got:  func(j *MapData) interface{} { return j.Info.IdTechnique },
		},
		"Tools.Config.Site": {
			want: "rpcache-aa",
			got:  func(j *MapData) interface{} { return j.Tools.Config.Site },
		},
		"Tools.Config.BaseUrl": {
			want: "meteofrance.com/internet2018client/2.0",
			got:  func(j *MapData) interface{} { return j.Tools.Config.BaseUrl },
		},
		"ChildrenPOI": {
			want: "VILLE_FRANCE",
			got:  func(j *MapData) interface{} { return j.Children[0].Taxonomy },
		},
		"Subzone": {
			want: Subzone{
				Path: "/previsions-meteo-france/auvergne-rhone-alpes/10",
				Name: "Auvergne-Rhône-Alpes",
			},
			got: func(j *MapData) interface{} { return j.Subzones["REGIN10"] },
		},
	}
	data := testMapParser(t, fileJsonRacine)
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
	f := testutils.OpenFile(t, fileJsonFilterFail)
	_, err := mapParser(f)
	if err == nil {
		t.Error("error expected")
		return
	}
}

func testMapParser(t *testing.T, file string) *MapData {
	f := testutils.OpenFile(t, file)
	defer f.Close()
	data, err := mapParser(f)
	if err != nil {
		t.Fatalf("mapParser(%s) error: %v", file, err)
	}
	return data
}

func TestName(t *testing.T) {
	m := MfMap{
		Data: testMapParser(t, fileJsonRacine),
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
