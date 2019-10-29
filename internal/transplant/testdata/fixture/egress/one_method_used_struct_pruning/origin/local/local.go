package local

import "origin.tld/user/proj/dep1"

func main() {
	t := dep1.ExportedType1{}
	t.UsedMethod1()
}
