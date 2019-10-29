package proj

import "copy.tld/user/proj/internal/dep1"

func main() {
	_ = dep1.ExportedType1{}
}
