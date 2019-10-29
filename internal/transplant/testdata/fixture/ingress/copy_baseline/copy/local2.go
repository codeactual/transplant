package proj

const (
	RewrittenLocalImportPath = "copy.tld/user/proj"
	RewrittenDepImportPath   = "copy.tld/user/proj/internal/dep1"
)

func FromUnchanged2() {
	_ = "FromUnchanged2"
}

func FromChanged2() {
	_ = "FromChanged2 (edit)"
}

func FromAdded2() {
	_ = "FromAdded2"
}
