package use

import (
	other_pkg "fixture.tld/use_and_shadowing/pkglocal/type/use"
)

func init() {
	var a other_pkg.ExportedType2
	_ = a
}

type ExportedType4Named other_pkg.ExportedType4

type ExportedType4NamedTwice ExportedType4Named

type ExportedType4NamedThrice ExportedType4NamedTwice

type ExportedAliasedIntAlias = other_pkg.ExportedAliasedInt

type ExportedAliasedIntAliasTwice = ExportedAliasedIntAlias

type ExportedAliasedIntAliasThrice = ExportedAliasedIntAliasTwice

var ExportedVar1 other_pkg.ExportedType4NamedThrice = 1

var ExportedVar2 other_pkg.ExportedAliasedIntAliasThrice = 2

const ExportedConst1 = other_pkg.ExportedConst1

const ExportedConst2 = other_pkg.ExportedConst2

const ExportedConst3 other_pkg.ExportedType4NamedThrice = 10

const ExportedConst4 other_pkg.ExportedAliasedIntAliasThrice = 20

const (
	ExportedConst5 other_pkg.ExportedType4NamedThrice      = 30
	ExportedConst6 other_pkg.ExportedAliasedIntAliasThrice = 40
)

var ExportedVar3 int = int(other_pkg.ExportedConst1)

var ExportedVar4 int = int(other_pkg.ExportedConst2)

var (
	ExportedVar5 = int(other_pkg.ExportedConst1)
	ExportedVar6 = int(other_pkg.ExportedConst2)
)

func InCall() {
	InCallee(other_pkg.ExportedType2{})
}

func InCallee(a other_pkg.ExportedType2) {
	b := a
	_ = b
}

func InSwitchCase() {
	var i interface{}
	switch i.(type) {
	case *other_pkg.ExportedType1:
	}

	var v other_pkg.ExportedType2
	switch v {
	case other_pkg.ExportedType2{}:
	}
}

func InAssign() {
	var a interface{} = other_pkg.ExportedType2{}
	_ = a
}

func InAssignShort() {
	a := other_pkg.ExportedType1{}
	b, c := other_pkg.ExportedType2{}, other_pkg.ExportedType3{}
	_, _, _ = a, b, c
}

func InCompositeLit() {
	type T struct {
		s other_pkg.ExportedType1
	}
	t1 := T{s: other_pkg.ExportedType1{}}

	t2 := struct {
		s other_pkg.ExportedType2
	}{
		s: other_pkg.ExportedType2{},
	}

	_, _ = t1, t2
}

func InSingleReturn() other_pkg.ExportedType1 {
	return other_pkg.ExportedType1{}
}

func InMultiReturn() (other_pkg.ExportedType1, other_pkg.ExportedType2) {
	return other_pkg.ExportedType1{}, other_pkg.ExportedType2{}
}

func InSingleTypeListFuncDecl(v other_pkg.ExportedType2) *other_pkg.ExportedType1 {
	return nil
}

func InMultiTypeListFuncDecl(x other_pkg.ExportedType3, y other_pkg.ExportedType4) (*other_pkg.ExportedType1, *other_pkg.ExportedType2) {
	return nil, nil
}

type inMethodDeclTypeList struct{}

func (t inMethodDeclTypeList) Single(v other_pkg.ExportedType2) *other_pkg.ExportedType1 {
	return nil
}

func (t inMethodDeclTypeList) Multi(x other_pkg.ExportedType3, y other_pkg.ExportedType4) (*other_pkg.ExportedType1, *other_pkg.ExportedType2) {
	return nil, nil
}

func InTypeAssertion() {
	var i interface{}
	_ = i.(other_pkg.ExportedType2)

	type hasMethod interface {
		Method1()
	}

	var _ hasMethod = (*other_pkg.ExportedType3)(nil)
}

func InTypeConversion() {
	type t struct{}

	_ = t(other_pkg.ExportedType1{})

	t2 := other_pkg.ExportedType2{}
	_ = t(t2)
}

func InAliasType() {
	type t1 = other_pkg.ExportedType1
	type t2 = map[string]other_pkg.ExportedType2
}

func InNamedType() {
	type t1 other_pkg.ExportedType1
	type t2 map[string]other_pkg.ExportedType2
}

func InMakeMap() {
	_ = make(map[string]other_pkg.ExportedType1)
	_ = make(map[other_pkg.ExportedType2]string)
}

func InMakeChan() {
	_ = make(chan other_pkg.ExportedType1)
}

type InInterfaceMethod interface {
	Method(other_pkg.ExportedType3, other_pkg.ExportedType4NamedThrice) other_pkg.ExportedAliasedIntAliasThrice
}

func PassInInterfaceMethodImpl() {
	UseInInterfaceMethodImpl(&other_pkg.InInterfaceMethodImpl{})
}

func UseInInterfaceMethodImpl(impl other_pkg.InInterfaceMethod) {
	impl.Method(other_pkg.ExportedType3{}, 0)
}

func UseHasAnonEmbeddedStruct() {
	s := other_pkg.HasAnonEmbeddedStruct{}
	s.NativeMethod()
}

func UseHasAnonEmbeddedStructMethod() {
	s := other_pkg.HasAnonEmbeddedStruct{}
	s.EmbeddedMethod()
}

func UseInterfaceAsVarType() {
	var i other_pkg.InInterfaceMethod
	i = &other_pkg.InInterfaceMethodImpl{}
	_ = i
}
