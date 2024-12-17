package static

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

var expectedFiles = map[string]struct {
	fs    fs.FS
	files []string
}{
	"js": {
		embedJS,
		[]string{"highcharts.js", "meteo.js"},
	},
	"css": {
		embedCSS,
		[]string{"meteo.css"},
	},
	"fonts": {
		embedFonts,
		[]string{"fa.woff2"},
	},
}

func TestStaticFiles(t *testing.T) {
	for dir, files := range expectedFiles {
		t.Run(dir, func(t *testing.T) {
			for _, f := range files.files {
				want := dir + "/" + f
				err := fstest.TestFS(files.fs, want)
				if err != nil {
					t.Error(err)
				}
			}
		})
	}
}
