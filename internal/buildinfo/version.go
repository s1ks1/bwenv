package buildinfo

import (
	"runtime/debug"
	"strings"
)

func Resolve(version string) string {
	if version != "" {
		return version
	}
	version = "v3.0.0-dev"
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	if v := bi.Main.Version; v != "" && !strings.HasPrefix(v, "(devel)") {
		version = v
		return version
	}
	var rev, modified string
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			modified = s.Value
		}
	}
	if len(rev) >= 7 {
		version = "v3.0.0-dev+" + rev[:7]
		if modified == "true" {
			version += "-dirty"
		}
	}
	return version
}
