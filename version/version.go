package version

const version = "3.1.2"

var (
	gitCommit = ""
	buildTime = ""
)

func Version() string {
	return version
}

func GitCommit() string {
	return gitCommit
}

func BuildTime() string {
	return buildTime
}
