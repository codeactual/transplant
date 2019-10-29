package proj

import "copy.tld/user/proj/internal/dep1"

func main() {
	t := dep1.ExportedType1{}
	t.UsedMethod1()
}
