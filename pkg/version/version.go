package version

// These variables are set at build time via -ldflags
var (
	gitVersion = "unknown"
	gitCommit  = "unknown"
)

// GetVersion returns the git version tag
func GetVersion() string {
	return gitVersion
}

// GetCommit returns the git commit hash
func GetCommit() string {
	return gitCommit
}
