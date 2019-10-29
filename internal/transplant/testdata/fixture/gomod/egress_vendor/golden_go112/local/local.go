package local

import (
	"github.com/mitchellh/mapstructure"

	"copy.tld/user/proj/internal"
)

func LocalFunc() {
	var c mapstructure.DecoderConfig
	_ = c
	internal.Dep1Func()
}
