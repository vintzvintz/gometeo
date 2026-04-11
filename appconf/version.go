package appconf

import "runtime/debug"

// CommitID is the git commit the binary was built from.
//
// Host builds: leave empty — Commit() reads vcs.revision that `go build`
// auto-embeds from the surrounding git tree.
//
// Container builds: must be injected at link time, because .dockerignore
// excludes .git from the build context so vcs.revision is unset inside
// the image:
//
//	go build -ldflags "-X gometeo/appconf.CommitID=${COMMIT_ID}"
var CommitID = ""

func Commit() string {
	if CommitID != "" {
		return CommitID
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				if len(s.Value) > 12 {
					return s.Value[:12]
				}
				return s.Value
			}
		}
	}
	return "unknown"
}
