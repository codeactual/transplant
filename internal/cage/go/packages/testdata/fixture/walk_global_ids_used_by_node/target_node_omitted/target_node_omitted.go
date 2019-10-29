package target_node_omitted

import "fixture.tld/walk_global_ids_used_by_node/target_node_omitted/dep1"

const Const = 1

const ConstAssignedWithImported = dep1.Const

var Var string = ""

var VarAssignedWithImported = dep1.Var

type Struct struct {
	S string
}

func (s Struct) Method() {
	dep1.Struct{}.Method()
}

type NamedType Struct

type NamedImportedType dep1.Struct

type AliasType = Struct

type AliasImportedType = dep1.Struct

func Func() {
	dep1.Func()
}
