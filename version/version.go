package version

const version = "4.0.2"

var (
	gitCommit = ""
	buildTime = ""
)

// Version return the version string
func Version() string {
	return version
}

// GitCommit return git commit on build
func GitCommit() string {
	return gitCommit
}

// BuildTime return build time
func BuildTime() string {
	return buildTime
}
