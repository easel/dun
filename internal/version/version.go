package version

import (
	"fmt"
	"runtime"
)

// Version information set via ldflags at build time.
// Example: go build -ldflags "-X github.com/easel/dun/internal/version.Version=1.0.0"
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// Info contains version information about the binary.
type Info struct {
	Version   string
	Commit    string
	BuildDate string
	GoVersion string
	Platform  string
}

// Get returns the current version information.
func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
}

// String returns a formatted version string.
func (i Info) String() string {
	return fmt.Sprintf("dun %s (%s) built %s with %s for %s",
		i.Version, i.Commit, i.BuildDate, i.GoVersion, i.Platform)
}
