package local

import "origin.tld/user/proj/dep5_used_by_local_impl"

func ExportedFunc1() {
	dep5_used_by_local_impl.ExportedFunc1()
}
