package baseline

const Const1 = 1

const Const2, Const3 = 2, 3

const (
	Const4 = 4
	Const5 = 5
)

var Var1 string = ""

var Var2, Var3 string = "", ""

var (
	Var4 = ""
	Var5 = ""
)

type Struct struct {
	S string
}

func (s Struct) Method() {
}

type (
	Type1 string
	Type2 string
)

type NamedType Struct

type AliasType = Struct

func Func() {
}
