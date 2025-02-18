package geojson_test

import (
	"fmt"
	gj "gometeo/geojson"
	"gometeo/testutils"
	"slices"
	"testing"
)

func TestParseMultiforecast(t *testing.T) {

	j := testutils.MultiforecastReader(t)
	defer j.Close()

	fc, err := gj.ParseMultiforecast(j)
	if err != nil {
		t.Fatal(fmt.Errorf("parseMultiforecast() error: %w", err))
	}
	if len(fc.Features) == 0 {
		t.Fatal("parseMultiforecast() returned no data")
	}
	f := fc.Features
	last := len(f[0].Properties.Forecasts) - 1
	if f[0].Properties.Forecasts[last].LongTerme == false {
		t.Error("long_terme has wrong value on last value")
	}
	if f[0].Properties.Forecasts[0].LongTerme == true {
		t.Error("long_terme has wrong value on first element")
	}
}

func TestPictoNames(t *testing.T) {
	j := testutils.MultiforecastReader(t)
	defer j.Close()

	const minLength = 20
	fc, err := gj.ParseMultiforecast(j)
	if err != nil {
		t.Error(err)
	}
	pics := fc.Features.PictoNames()
	if len(pics) < minLength {
		t.Errorf("picto list is too short (<%d items), %v", minLength, pics)
	}
	if slices.Contains[[]string, string](pics, "") {
		t.Errorf("picto list contains an empty string : %v", pics)
	}
}
