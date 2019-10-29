package use

type nonExportedType1 struct{}
type ExportedType1 struct{}
type ExportedType2 struct{}

type ExportedType3 struct{}

func (t ExportedType3) Method1() {}

type ExportedType4 int

type ExportedType4Named ExportedType4

type ExportedType4NamedTwice ExportedType4Named

type ExportedType4NamedThrice ExportedType4NamedTwice

type ExportedAliasedInt = int

type ExportedAliasedIntAlias = ExportedAliasedInt

type ExportedAliasedIntAliasTwice = ExportedAliasedIntAlias

type ExportedAliasedIntAliasThrice = ExportedAliasedIntAliasTwice

var ExportedVar1 ExportedType4NamedThrice = 1

var ExportedVar2 ExportedAliasedIntAliasThrice = 2

const ExportedConst1 ExportedType4NamedThrice = 10

const ExportedConst2 ExportedAliasedIntAliasThrice = 20

const (
	ExportedConst3 ExportedType4NamedThrice      = 30
	ExportedConst4 ExportedAliasedIntAliasThrice = 40
)

var ExportedVar3 int = int(ExportedConst1)

var ExportedVar4 int = int(ExportedConst2)

var (
	ExportedVar5 = int(ExportedConst1)
	ExportedVar6 = int(ExportedConst2)
)

func init() {
	var a nonExportedType1
	_ = a
}

func InCall() {
	InCallee(nonExportedType1{})
}

func InCallee(a nonExportedType1) {
	b := a
	_ = b
}

func InSwitchCase() {
	var i interface{}
	switch i.(type) {
	case *ExportedType1:
	}

	var v nonExportedType1
	switch v {
	case nonExportedType1{}:
	}
}

func InAssign() {
	var a interface{} = nonExportedType1{}
	_ = a
}

func InAssignShort() {
	a := ExportedType1{}
	b, c := ExportedType2{}, ExportedType3{}
	_, _, _ = a, b, c
}

func InCompositeLit() {
	type T struct {
		s ExportedType1
	}
	t1 := T{s: ExportedType1{}}

	t2 := struct {
		s ExportedType2
	}{
		s: ExportedType2{},
	}

	_, _ = t1, t2
}

func InSingleReturn() ExportedType1 {
	return ExportedType1{}
}

func InMultiReturn() (ExportedType1, ExportedType2) {
	return ExportedType1{}, ExportedType2{}
}

func InSingleTypeListFuncDecl(v ExportedType2) *ExportedType1 {
	return nil
}

func InMultiTypeListFuncDecl(x ExportedType3, y ExportedType4) (*ExportedType1, *ExportedType2) {
	return nil, nil
}

type inMethodDeclTypeList struct{}

func (t inMethodDeclTypeList) Single(v ExportedType2) *ExportedType1 {
	return nil
}

func (t inMethodDeclTypeList) Multi(x ExportedType3, y ExportedType4) (*ExportedType1, *ExportedType2) {
	return nil, nil
}

func InTypeAssertion() {
	var i interface{}
	_ = i.(nonExportedType1)

	type hasMethod interface {
		Method1()
	}

	var _ hasMethod = (*ExportedType3)(nil)
}

func InTypeConversion() {
	type t struct{}

	_ = t(ExportedType1{})

	t2 := ExportedType2{}
	_ = t(t2)
}

func InAliasType() {
	type t1 = ExportedType1
	type t2 = map[string]ExportedType2
}

func InNamedType() {
	type t1 ExportedType1
	type t2 map[string]ExportedType2
}

func InMakeMap() {
	_ = make(map[string]ExportedType1)
	_ = make(map[ExportedType2]string)
}

func InMakeChan() {
	_ = make(chan ExportedType1)
}

type InInterfaceMethod interface {
	Method(ExportedType3, ExportedType4NamedThrice) ExportedAliasedIntAliasThrice
}

type InInterfaceMethodImpl struct{}

func (i *InInterfaceMethodImpl) Method(ExportedType3, ExportedType4NamedThrice) ExportedAliasedIntAliasThrice {
	return 0
}

func PassInInterfaceMethodImpl() {
	UseInInterfaceMethodImpl(&InInterfaceMethodImpl{})
}

func UseInInterfaceMethodImpl(impl InInterfaceMethod) {
	impl.Method(ExportedType3{}, 0)
}

type embedded struct {
}

func (e embedded) EmbeddedMethod() {
}

type HasAnonEmbeddedStruct struct {
	embedded
}

func (s HasAnonEmbeddedStruct) NativeMethod() {
}

func UseHasAnonEmbeddedStruct() {
	s := HasAnonEmbeddedStruct{}
	s.NativeMethod()
}

func UseHasAnonEmbeddedStructMethod() {
	s := HasAnonEmbeddedStruct{}
	s.EmbeddedMethod()
}

func UseInterfaceAsVarType() {
	var i InInterfaceMethod
	i = &InInterfaceMethodImpl{}
	_ = i
}
