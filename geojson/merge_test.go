package geojson_test

import (
	"gometeo/testutils"
	"reflect"
	"testing"

	gj "gometeo/geojson"
)

func TestMergePrevList(t *testing.T) {
	// create a map
	old := testutils.BuildTestMap(t).Prevs
	new := testutils.BuildTestMap(t).Prevs

	// find oldest date
	var oldest gj.Date
	for date := range new {
		tmp := oldest.Sub(date)
		if (oldest == gj.Date{}) || tmp > 0 {
			oldest = date
		}
	}

	// remove daily and nuit, both should be present at any time.
	pad := new[oldest]
	delete(pad, gj.Nuit)
	delete(pad, gj.Journalier)
	new[oldest] = pad

	// merge with original
	new.Merge(old, -1000, 1000)

	// check new has old map restored
	pad = new[oldest]
	_, ok1 := pad[gj.Journalier]
	_, ok2 := pad[gj.Nuit]
	if !ok1 || !ok2 {
		t.Errorf("missing moments in merged PrevList")
	}
}


func TestMergeChroniques(t *testing.T) {

	// prepare two prevlists
	old := testutils.BuildTestMap(t).Graphdata
	new := testutils.BuildTestMap(t).Graphdata
	if !reflect.DeepEqual(new, old) {
		t.Error("initial prevlists should be equal")
	}


	// iterate over new to remove some points
	const nbRemoved = 3
	for nom := range new {
		serie := new[nom]
		for insee := range serie {
			chro := serie[insee]
			serie[insee] = chro[nbRemoved:]
		}
		new[nom] = serie
	}
	if reflect.DeepEqual(new, old) {
		t.Error("initial prevlists should not be equal after trucation")
	}

	// call method under test
	new.Merge(old, -1000, +1000)
	if !reflect.DeepEqual(new, old) {
		t.Error("prevlists should be equal after merge")
	}
}
