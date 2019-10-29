package file

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// CopyFile copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file. The file mode will be copied from the source and
// the copied data is synced/flushed to stable storage.
//
// MIT License
// Copyright (c) 2017 Roland Singer [roland.singer@desertbit.com]
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//
// Origin:
//   (original) https://gist.github.com/m4ng0squ4sh/92462b38df26839a3ca324697c8cba04
//   (now at) https://gist.github.com/r0l1/92462b38df26839a3ca324697c8cba04
//   https://github.com/r0l1
//
// Changes:
//
//   - Remove naked returns.
//   - Wrap errors.
//   - Fix missing error propogation.
func CopyFile(src, dst string) (err error) {
	in, err := os.Open(src) // #nosec G304
	if err != nil {
		return errors.Wrapf(err, "failed to open source file [%s] for copy", src)
	}
	defer func() { err = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return errors.Wrapf(err, "failed to create destination file [%s] for copy", dst)
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return errors.Wrapf(err, "failed to complete copy I/O from file [%s] to [%s]", src, dst)
	}

	err = out.Sync()
	if err != nil {
		return errors.Wrapf(err, "failed to sync copy from file [%s] to [%s]", src, dst)
	}

	si, err := os.Stat(src)
	if err != nil {
		return errors.Wrapf(err, "failed to stat copy source file [%s]", src)
	}
	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return errors.Wrapf(err, "failed to set permissions of copy destination file [%s]", dst)
	}

	return nil
}

// CopyDir recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist, destination directory must *not* exist.
// Symlinks are ignored and skipped.
//
// MIT License
// Copyright (c) 2017 Roland Singer [roland.singer@desertbit.com]
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//
// Origin:
//   (original) https://gist.github.com/m4ng0squ4sh/92462b38df26839a3ca324697c8cba04
//   (now at) https://gist.github.com/r0l1/92462b38df26839a3ca324697c8cba04
//   https://github.com/r0l1
//
// Changes:
//
//   - Remove naked returns.
//   - Wrap errors.
//   - Fix missing error propogation.
func CopyDir(src string, dst string) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return errors.Wrapf(err, "failed to stat copy source dir [%s]", src)
	}
	if !si.IsDir() {
		return errors.Errorf("copy source is not a dir [%s]", src)
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to stat copy destination dir [%s]", dst)
	}
	if err == nil {
		return errors.Errorf("destination already exists [%s]", dst)
	}

	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return errors.Wrapf(err, "failed to make all copy destination dirs [%s]", dst)
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return errors.Wrapf(err, "failed to read source dir [%s]", src)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDir(srcPath, dstPath)
			if err != nil {
				return errors.Wrapf(err, "failed to copy dir [%s] to [%s]", srcPath, dstPath)
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = CopyFile(srcPath, dstPath)
			if err != nil {
				return errors.Wrapf(err, "failed to copy file [%s] to [%s]", srcPath, dstPath)
			}
		}
	}

	return nil
}
