package local2

import "origin.tld/user/proj/dep1"

func NamedTypeChain() {
	type t dep1.NamedTypeChain
}

func AliasTypeChain() {
	type t = dep1.AliasTypeChain
}

func FieldTypeChain() {
	type t struct {
		f dep1.FieldTypeChain
	}
}
