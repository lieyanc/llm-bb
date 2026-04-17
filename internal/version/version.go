package version

import "runtime"

var (
	Version   = "dev"
	Commit    = "unknown"
	Channel   = "dev"
	BuildDate = "unknown"
)

type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Channel   string `json:"channel"`
	BuildDate string `json:"buildDate"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
}

func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Channel:   Channel,
		BuildDate: BuildDate,
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
	}
}

func ShortCommit() string {
	if len(Commit) < 7 {
		return Commit
	}
	return Commit[:7]
}

// Injected reports whether ldflags wrote real build metadata in.
func Injected() bool {
	return Commit != "unknown"
}
