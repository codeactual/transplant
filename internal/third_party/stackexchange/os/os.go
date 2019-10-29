package os

import (
	std_os "os"

	"github.com/pkg/errors"
)

// Origin:
//   https://stackoverflow.com/questions/22744443/check-if-there-is-something-to-read-on-stdin-in-golang/26567513#26567513
//   https://stackoverflow.com/users/571904/ostler-c
//
// Changes:
//   - Migrate to github.com/pkg/errors
func IsPipeStdin() (bool, error) {
	stat, err := std_os.Stdin.Stat()
	if err != nil {
		return false, errors.WithStack(err)
	}
	return (stat.Mode() & std_os.ModeCharDevice) == 0, nil
}
