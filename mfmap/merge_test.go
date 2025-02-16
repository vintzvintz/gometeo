package mfmap_test

import (
	"gometeo/mfmap"
	"testing"
)

// remove nb items from the time-series on each (geo)feature
func truncateTimeSeries(m *mfmap.MfMap, nbForecast int, nbDaily int) {
	for i := range m.Multi {
		m.Multi[i].Properties.Forecasts = m.Multi[i].Properties.Forecasts[nbForecast:]
		m.Multi[i].Properties.Dailies = m.Multi[i].Properties.Dailies[nbDaily:]
	}
}

func countItems(m *mfmap.MfMap) (nbFeatures, nbForecasts, nbDailies int) {
	for i := range m.Multi {
		// forecast count for current (geo)feature
		nbForecasts += len(m.Multi[i].Properties.Forecasts)
		nbDailies += len(m.Multi[i].Properties.Dailies)
	}
	return len(m.Multi), nbForecasts, nbDailies
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
