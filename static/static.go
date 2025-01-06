package static

import (
	"embed"
	"net/http"
)

const (
	Js    = "/js/"
	Css   = "/css/"
	Fonts = "/fonts/"
)

//go:embed js
var embedJS embed.FS

//go:embed css
var embedCSS embed.FS

//go:embed fonts
var embedFonts embed.FS

func Register(mux *http.ServeMux) {
	mux.Handle(Js, http.FileServerFS(embedJS))
	mux.Handle(Css, http.FileServerFS(embedCSS))
	mux.Handle(Fonts, http.FileServerFS(embedFonts))
}
