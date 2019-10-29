package local

import "origin.tld/user/proj/dep1"

func main() {
	_ = dep1.ExportedType1{}
}
