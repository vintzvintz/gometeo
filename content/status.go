package content

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"gometeo/appconf"
	"gometeo/mfmap"
)

//go:embed status.html
var statusTemplate string

// displayLoc is the zone used to render wall-clock timestamps on the status
// page. Pinned to Europe/Paris so the page looks the same whether the server
// runs in a UTC Docker container or on a developer box. Falls back to UTC if
// the tzdata is unavailable.
var displayLoc = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		return time.UTC
	}
	return loc
}()

type Stats struct {
	Name       string
	Path       string
	LastUpdate string
	LastHit    string
	NextUpdate string
	UpdateMode string
	HitCount   int64
}

// ReportView is a template-friendly (pre-formatted strings) flattening of
// StatusReport. Keeps the HTML template dumb.
type ReportView struct {
	Uptime           string
	StartTime        string
	Commit           string
	NextUpdatable    string
	UpstreamRequests int64
	StaticServed     int64
	Counters         CountersView
	RecentErrors     []ErrorRow
}

// CountersView holds the per-resource loaded/failed/served counts
// plus the totals row. Upstream requests are tracked separately at
// the client level — see ReportView.UpstreamRequests.
type CountersView struct {
	Maps   CounterRow
	Pictos CounterRow
	Total  CounterRow
}

type CounterRow struct {
	Loaded int64
	Failed int64
	Served int64
}

// ErrorRow is the display form of an obs.ErrorEvent.
type ErrorRow struct {
	Time   string
	Age    string
	Source string
	Target string
	Err    string
}

func (mc *Meteo) buildReportView() ReportView {
	r := mc.Report()
	maps := CounterRow{
		Loaded: int64(r.MapsLoaded),
		Failed: r.Obs.MapsFailed,
		Served: r.Obs.MapsServed,
	}
	pictos := CounterRow{
		Loaded: int64(r.PictosLoaded),
		Failed: r.Obs.PictosFailed,
		Served: r.Obs.PictosServed,
	}
	total := CounterRow{
		Loaded: maps.Loaded + pictos.Loaded,
		Failed: maps.Failed + pictos.Failed,
		Served: maps.Served + pictos.Served,
	}
	rv := ReportView{
		Uptime:           r.Obs.Uptime.Round(time.Second).String(),
		StartTime:        r.Obs.StartTime.In(displayLoc).Format("2006-01-02 15:04:05 MST"),
		Commit:           appconf.Commit(),
		NextUpdatable:    r.NextUpdatable,
		UpstreamRequests: r.Obs.UpstreamRequests,
		StaticServed:     r.Obs.StaticServed,
		Counters: CountersView{
			Maps:   maps,
			Pictos: pictos,
			Total:  total,
		},
	}
	for _, e := range r.Obs.RecentErrors {
		rv.RecentErrors = append(rv.RecentErrors, ErrorRow{
			Time:   e.Time.Format("15:04:05"),
			Age:    time.Since(e.Time).Round(time.Second).String(),
			Source: string(e.Source),
			Target: e.Target,
			Err:    e.Err,
		})
	}
	return rv
}

func getStats(m *mfmap.MfMap) Stats {
	s := Stats{
		Name:       m.Name(),
		Path:       m.Path(),
		HitCount:   m.Schedule.HitCount(),
		UpdateMode: "-",
		LastHit:    "-",
		LastUpdate: "-",
		NextUpdate: "-",
	}
	if lh := m.Schedule.LastHit(); !lh.IsZero() {
		s.LastHit = time.Since(lh).Round(time.Second).String()
	}
	if lu := m.Schedule.LastUpdate(); !lu.IsZero() {
		s.LastUpdate = time.Since(lu).Round(time.Second).String()
		s.NextUpdate = m.Schedule.DurationToUpdate().Round(time.Second).String()
	}
	if m.Schedule.IsHot() {
		s.UpdateMode = "hot"
	}
	return s
}

func (s Stats) String() string {
	return fmt.Sprintf("%s mode:%s lastUpdate:%v lastHit:%v hitCount:%d\n",
		s.Name, s.UpdateMode, s.LastUpdate, s.LastHit, s.HitCount)
}

func (ms *mapStore) Status() []Stats {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	// sort keys by name for displaying maps in constant ordre
	// TODO : use stdlib functions available in go 1.23
	var names = make([]string, 0, len(ms.store))
	for k := range ms.store {
		names = append(names, k)
	}
	slices.Sort(names)

	stats := make([]Stats, 0, len(ms.store))
	for _, name := range names {
		m := ms.store[name]
		stats = append(stats, getStats(m))
	}
	return stats
}

func (mc *Meteo) makeStatusHandler() http.HandlerFunc {
	// compile template only once
	tmpl, err := template.New("").Parse(statusTemplate)
	if err != nil {
		panic(err) // error compiling status page template
	}
	// return closure having http.HandlerFunc signature
	return func(resp http.ResponseWriter, _ *http.Request) {
		d := struct {
			Report ReportView
			Stats  []Stats
		}{
			Report: mc.buildReportView(),
			Stats:  mc.maps.Status(),
		}
		b := &bytes.Buffer{}
		err := tmpl.Execute(b, d)
		if err != nil {
			slog.Error("statusHandler error", "err", err)
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}
		resp.Header().Add("X-Robots-Tag", "noindex, nofollow")
		resp.WriteHeader(http.StatusOK)
		_, err = io.Copy(resp, b)
		if err != nil {
			slog.Error("send error", "err", err)
		}
	}
}
