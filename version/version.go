package version

const version = "3.1.4"

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
