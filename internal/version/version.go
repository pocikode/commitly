package version

// These values are injected at build time via -ldflags, e.g.:
//
//	go build -ldflags "-X github.com/pocikode/commitly/internal/version.Version=v1.2.3"
var (
	// Version is the semantic version of the build.
	Version = "dev"
	// Commit is the git commit hash of the build.
	Commit = "none"
	// Date is the build date.
	Date = "unknown"
)

// String returns a human-readable version string.
func String() string {
	return Version + " (commit " + Commit + ", built " + Date + ")"
}
