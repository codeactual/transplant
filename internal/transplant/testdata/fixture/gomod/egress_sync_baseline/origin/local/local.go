package local

import (
	"github.com/mitchellh/mapstructure"

	"origin.tld/user/proj/dep1"
)

func LocalFunc() {
	var c mapstructure.DecoderConfig
	_ = c
	dep1.Dep1Func()
}
