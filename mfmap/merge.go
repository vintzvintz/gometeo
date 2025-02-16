package mfmap

import "slices"

func (m *MfMap) MergeOld(old *MfMap, pastDays int) {
	// preserve stats - lastUpdate excluded
	m.stats.hitCount.Store(old.HitCount())
	m.stats.lastHit.Store(old.LastHit())

	// retreive parent map name from old map because it is only available
	// at initial recursive fetch and not when updating individual map
	m.Parent = old.Parent

	// temp hashmaps for lookup on old (geo)features
	oldForecasts := make(map[codeInsee][]Forecast, len(old.Multi))
	oldDailies := make(map[codeInsee][]Daily, len(old.Multi))

	for _, feat := range old.Multi {
		oldForecasts[feat.Properties.Insee] = feat.Properties.Forecasts
		oldDailies[feat.Properties.Insee] = feat.Properties.Dailies
	}

	// iterate over (geo)features
	// merge both inputs (old and new []Forecast series) into new time-series
	for i := range m.Multi {
		mergedF := mergeTimeSeries(
			oldForecasts[m.Multi[i].Properties.Insee],
			m.Multi[i].Properties.Forecasts,
			pastDays)
		m.Multi[i].Properties.Forecasts = mergedF

		mergedD := mergeTimeSeries(
			oldDailies[m.Multi[i].Properties.Insee],
			m.Multi[i].Properties.Dailies,
			pastDays)
		m.Multi[i].Properties.Dailies = mergedD
	}
}

// merge two []Daily or []Forecasts
// new overwrites old on a same echeance
// old and new are not mutated, a new slice is allocated for the result
// pastDays sets limit on history retention. negative value = no limit
func mergeTimeSeries[T Echeancer](old []T, new []T, pastDays int) []T {

	// use a temp map keyed by echeance to merge both time-series
	tmp := make(map[Echeance]T)

	insert := func(serie []T) {
		for _, prev := range serie {
			ech := prev.Echeance()
			if (pastDays < 0) || -pastDays < int(ech.Date.DaysFromNow()) {
				tmp[ech] = prev
			}
		}
	}
	// order is important, newer data overwrites old data
	insert(old)
	insert(new)

	// sort echeances
	echs := make([]Echeance, 0, len(tmp))
	for k := range tmp {
		echs = append(echs, k)
	}
	slices.SortFunc(echs, CompareEcheances)

	// TODO: use stdlib 'maps' package on go 1.23
	merged := make([]T, 0, len(tmp))
	for _, ech := range echs {
		merged = append(merged, tmp[ech])
	}
	return merged
}
