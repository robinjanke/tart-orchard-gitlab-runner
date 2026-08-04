package version

var (
	Version = "dev"
	Commit  = "unknown"
)

func FullVersion() string {
	return Version + "+" + Commit
}
