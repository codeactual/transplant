package ldflags

// ldflags receives build-time variable assignments.
//
// Update LDFLAGS in Makefile to manage them.

// Version is value like "8107551-master(-dirty)".
var Version string

func init() {
	if Version == "" {
		Version = "unknown (ldflags.Version not set via -ldflags at build time)"
	}
}
