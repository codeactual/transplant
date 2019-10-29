package local

const (
	RewrittenLocalImportPath = "origin.tld/user/proj/local"
	RewrittenDepImportPath   = "origin.tld/user/proj/dep1"
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
