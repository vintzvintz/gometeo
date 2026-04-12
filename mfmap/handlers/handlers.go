package handlers

import (
	"bytes"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"gometeo/mfmap"
	"gometeo/obs"
)

// Register adds handlers to mux for "/$path", "/$path/data", "/$path/cacheid/svg"
// and a redirection from "/" to "/france". reg may be nil (observability disabled).
func Register(mux *http.ServeMux, m *mfmap.MfMap, reg *obs.Registry) {
	p := "/" + m.Path()
	mux.HandleFunc(p, makeMainHandler(m))
	mux.HandleFunc(p+"/data", makeDataHandler(m, reg))
	mux.HandleFunc(p+"/"+m.Conf.CacheId+"/svg", makeSvgMapHandler(m))
	if p == "/france" {
		mux.HandleFunc("/{$}", makeRedirectHandler("/france"))
	}
}

func makeMainHandler(m *mfmap.MfMap) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		buf := bytes.Buffer{}
		err := WriteHtml(&buf, m)
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			slog.Error("BuildHtml error", "url", req.URL, "err", err)
			return
		}
		resp.Header().Add("Content-Type", "text/html; charset=utf-8")
		resp.Header().Add("Cache-Control", "no-cache")
		resp.WriteHeader(http.StatusOK)
		_, err = io.Copy(resp, &buf)
		if err != nil {
			slog.Error("send error", "err", err)
		}
	}
}

func makeDataHandler(m *mfmap.MfMap, reg *obs.Registry) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		buf := bytes.Buffer{}
		err := WriteJson(&buf, m)
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			slog.Error("BuildJson error", "url", req.URL, "err", err)
			return
		}
		resp.Header().Add("Content-Type", "application/json")
		resp.Header().Add("Cache-Control", "no-cache")
		resp.WriteHeader(http.StatusOK)
		_, err = io.Copy(resp, &buf)
		if err != nil {
			slog.Error("send error", "err", err)
		}
		// update on data handler (JSON request) instead of main handler
		// to allow main page caching and avoid simplest bots
		m.Schedule.MarkHit(clientIP(req))
		reg.RecordMapServed()
	}
}

func makeSvgMapHandler(m *mfmap.MfMap) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		if len(m.SvgMap) == 0 {
			resp.WriteHeader(http.StatusNotFound)
			slog.Warn("SVG map unavailable", "url", req.URL)
			return
		}
		resp.Header().Add("Cache-Control", "max-age=31536000, immutable")
		resp.Header().Add("Content-Type", "image/svg+xml")
		resp.WriteHeader(http.StatusOK)
		_, err := io.Copy(resp, bytes.NewReader(m.SvgMap))
		if err != nil {
			slog.Error("send error", "err", err)
		}
	}
}

// clientIP returns the real client IP, looking through reverse-proxy headers.
func clientIP(req *http.Request) string {
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For may contain a chain: "client, proxy1, proxy2"
		if ip, _, ok := strings.Cut(xff, ","); ok {
			return strings.TrimSpace(ip)
		}
		return strings.TrimSpace(xff)
	}
	if xri := req.Header.Get("X-Real-Ip"); xri != "" {
		return xri
	}
	ip, _, _ := net.SplitHostPort(req.RemoteAddr)
	return ip
}

func makeRedirectHandler(url string) func(http.ResponseWriter, *http.Request) {
	return func(resp http.ResponseWriter, req *http.Request) {
		slog.Info("redirect", "from", req.URL, "to", url)
		http.Redirect(resp, req, url, http.StatusMovedPermanently)
	}
}
