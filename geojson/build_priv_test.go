package geojson
/*
import (
	"fmt"
	"testing"
	"time"

	//"gometeo/testutils"
)

func TestFindDaily(t *testing.T) {
	//mf := makeMultiforecast(t)

	j := testutils.MultiforecastReader(t)
	defer j.Close()

	fc, err := ParseMultiforecast(j)
	if err != nil {
		t.Error(fmt.Errorf("parseMfCollection() error: %w", err))
	}
	if len(fc.Features) == 0 {
		t.Fatal("parseMfCollection() returned no data")
	}
	mf := fc.Features


	// these values must be updated after a change in test_data...
	id := codeInsee("751010") // "name": "Paris—1er Arrondissement"
	ech, err := time.Parse(time.RFC3339, "2025-01-02T00:00:00.000Z")
	if err != nil {
		t.Fatal(err)
	}
	d := mf.findDaily(id, Echeance{Date:NewDate(ech)})
	if d == nil {
		t.Fatalf("FindDaily() did not found daily forecast for location '%s' at '%s'", id, ech)
	}
}
	*/
