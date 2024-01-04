package version

var (
	version = "dev" // this will be the release tag
	gitSHA  = "dev" // sha1 from git, output of $(git rev-parse HEAD)
)

func Version() string {
	return version
}

func GitSHA() string {
	return gitSHA
}
