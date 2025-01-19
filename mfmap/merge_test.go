package mfmap_test

import (
	"gometeo/mfmap"
	"testing"
)

// remove nb items from the time-series on each (geo)feature
func truncateTimeSeries(m *mfmap.MfMap, nbForecast int, nbDaily int) {
	for i := range m.Forecasts {
		m.Forecasts[i].Properties.Forecasts = m.Forecasts[i].Properties.Forecasts[nbForecast:]
		m.Forecasts[i].Properties.Dailies = m.Forecasts[i].Properties.Dailies[nbDaily:]
	}
}

func countItems(m *mfmap.MfMap) (nbFeatures, nbForecasts, nbDailies int) {
	for i := range m.Forecasts {
		// forecast count for current (geo)feature
		nbForecasts += len(m.Forecasts[i].Properties.Forecasts)
		nbDailies += len(m.Forecasts[i].Properties.Dailies)
	}
	return len(m.Forecasts), nbForecasts, nbDailies
}

func TestMerge(t *testing.T) {
	// create a map
	old := testBuildMap(t)
	oldFeats, oldForecasts, oldDailies := countItems(old)

	// truncate timeseries
	new := testBuildMap(t)
	truncateTimeSeries(new, 3, 1)
	trFeats, trForecasts, trDailies := countItems(new)

	// merge with original
	new.MergeOld(old, -1)
	mergeFeats, mergeForecasts, mergeDailies := countItems(new)

	// check number of items
	if (mergeFeats != oldFeats) || (trFeats != oldFeats) {
		t.Errorf("number of (geo)features should be constant")
		t.Errorf("got %d truncate->%d merge->%d", oldFeats, trFeats, mergeFeats)
	}
	if (mergeForecasts != oldForecasts) || (trForecasts >= oldForecasts) {
		t.Errorf("inconsistent number of elements in forecast time-series")
		t.Errorf("got %d truncate->%d merge->%d", oldForecasts, trForecasts, trDailies)
	}
	if (mergeDailies != oldDailies) || (trDailies >= oldDailies) {
		t.Errorf("inconsistent number of dailies ")
		t.Errorf("got %d truncate->%d merge->%d", oldDailies, trDailies, mergeDailies)
	}
}
