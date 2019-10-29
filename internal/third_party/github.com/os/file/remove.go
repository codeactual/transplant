package file

import (
	"os"
	"path/filepath"
)

// UnsafeRemoveAll removes a directory written by Download or Unzip, first applying
// any permission changes needed to do so.
//
// Origin:
//   https://github.com/golang/go/blob/go1.12.2/src/cmd/go/internal/modfetch/unzip.go#L161
//   https://golang.org/LICENSE
//
// Changes:
//   - Allow a Chmod error to end the walk.
func UnsafeRemoveAll(dir string) error {
	// Module cache has 0555 directories; make them writable in order to remove content.
	walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // ignore errors walking in file system
		}
		if !info.IsDir() {
			return nil
		}
		return os.Chmod(path, 0777) // #nosec
	})
	if walkErr != nil {
		return walkErr
	}
	return os.RemoveAll(dir)
}
