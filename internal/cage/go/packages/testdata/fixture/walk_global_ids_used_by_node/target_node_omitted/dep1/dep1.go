package dep1

const Const = 1

var Var string = ""

type Struct struct {
	S string
}

func (s Struct) Method() {
}

type NamedType Struct

type AliasType = Struct

func Func() {
}
