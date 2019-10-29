package local

import (
	"path/filepath"
	fp "path/filepath"

	"origin.tld/user/proj/dep1"
)

func LocalFunc() {
	dep1.Dep1Func()
	_ = filepath.Separator
	_ = fp.Separator
}
