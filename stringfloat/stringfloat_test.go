package stringfloat

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestStringFloat(t *testing.T) {
	type Item struct {
		A StringFloat `json:"a"`
		B StringFloat `json:"b"`
		C StringFloat `json:"c"`
		D StringFloat `json:"d"`
	}
	jsonData := []byte(`{"a":null, "c":"51", "d":51}`) // b is missing
	want := Item{A: 0, B: 0, C: 51, D: 51}
	var item Item
	err := json.Unmarshal(jsonData, &item)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(item, want) {
		t.Errorf("StringFloat custom Unmarshall() got %v, want %v", item, want)
	}
}
