package file

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	cage_io "github.com/codeactual/transplant/internal/cage/io"
	cage_file "github.com/codeactual/transplant/internal/cage/os/file"
)

// RemoveContents removes all contents of a directory but keeps the directory itself.
//
// Origin:
//   https://stackoverflow.com/questions/33450980/golang-remove-all-contents-of-a-directory/33451503#33451503
//   https://stackoverflow.com/users/221700/peterso
//
// Changes:
//   - Migrate to cage/io.CloseOrStderr
//   - Migrate to cage/os/file.RemoveAllSafer
//   - Migrate to cage/os/file.ValidateRemove
//   - Migrate to github.com/pkg/errors
func RemoveContents(dir string) error {
	if err := cage_file.ValidateRemove(dir); err != nil {
		return errors.WithStack(err)
	}

	d, err := os.Open(dir) // #nosec G304
	if err != nil {
		return errors.Wrapf(err, "failed to open [%s]", dir)
	}
	defer cage_io.CloseOrStderr(d, dir)
	names, err := d.Readdirnames(-1)
	if err != nil {
		return errors.Wrapf(err, "failed to readdir [%s]", dir)
	}
	for _, name := range names {
		target := filepath.Join(dir, name)
		err = cage_file.RemoveAllSafer(target)
		if err != nil {
			return errors.Wrapf(err, "failed to remove all [%s]", target)
		}
	}
	return nil
}
