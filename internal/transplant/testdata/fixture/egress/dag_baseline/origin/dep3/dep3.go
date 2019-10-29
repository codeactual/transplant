package dep3

const ConstChain = 10

type NamedTypeChain string

type AliasTypeChain int

type FieldTypeChain bool

type ExportedType1 struct{}

func (t ExportedType1) MethodCallChainFromCompositeLit() {
}

func (t ExportedType1) MethodCallChainFromValue() {
}

func FuncCallChain() {
}
