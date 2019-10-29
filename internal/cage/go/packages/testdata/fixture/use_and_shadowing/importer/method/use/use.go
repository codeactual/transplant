package use

import (
	other_pkg "fixture.tld/use_and_shadowing/pkglocal/method/use"
)

func init() {
	a := other_pkg.ExportedType2{}.Method1
	b := other_pkg.ExportedType1{}.Method1()
	_, _ = a, b

	c := other_pkg.ExportedType3{}
	c.Method1()
}

func InCallAsValue() {
	f := func(v func() string) {}

	f(other_pkg.ExportedType2{}.Method1)

	t1 := other_pkg.ExportedType1{}
	f(t1.Method1)
}

func InCallForReturn() {
	f := func(v string) {}

	f(other_pkg.ExportedType2{}.Method1())

	t1 := other_pkg.ExportedType3{}
	f(t1.Method1())
}

func InIfForReturn() {
	if (other_pkg.ExportedType2{}).Method1() == "" {
	}

	t1 := other_pkg.ExportedType1{}
	if t1.Method1() == "" {
	}
}

func InSwitchForReturn() {
	switch (other_pkg.ExportedType2{}).Method1() {
	}

	t1 := other_pkg.ExportedType3{}
	switch t1.Method1() {
	}
}

func InSwitchCaseForReturn() {
	var a string
	switch a {
	case other_pkg.ExportedType2{}.Method1():
	}

	var b string
	t1 := other_pkg.ExportedType1{}
	switch b {
	case t1.Method1():
	}
}

func InSelectCaseForReturn() {
	select {
	case <-other_pkg.ExportedType4{}.Method1():
	}
}

func InSelectCaseAssignForReturn() {
	select {
	case f := <-other_pkg.ExportedType4{}.Method1():
		_ = f
	}
}

func InAssignAsValue() {
	var a interface{} = other_pkg.ExportedType2{}.Method1
	_ = a

	t1 := other_pkg.ExportedType3{}
	var b interface{} = t1.Method1
	_ = b
}

func InAssignShortAsValue() {
	a := other_pkg.ExportedType1{}.Method1
	b, c := other_pkg.ExportedType2{}.Method1, other_pkg.ExportedType3{}.Method1
	_, _, _ = a, b, c
}

func InAssignForReturn() {
	var a string = other_pkg.ExportedType2{}.Method1()
	_ = a

	t1 := other_pkg.ExportedType1{}
	var b string = t1.Method1()
	_ = b
}

func InAssignShortForReturn() {
	a := other_pkg.ExportedType1{}.Method1()
	b, c := other_pkg.ExportedType2{}.Method1(), other_pkg.ExportedType3{}.Method1()
	_, _, _ = a, b, c
}

func InCompositeLitAsValue() {
	type T struct {
		f func() string
	}
	t1 := T{f: other_pkg.ExportedType1{}.Method1}

	t2 := struct {
		f func() string
	}{
		f: other_pkg.ExportedType2{}.Method1,
	}

	_, _ = t1, t2
}

func InCompositeLitForReturn() {
	type T struct {
		s string
	}
	t1 := T{s: other_pkg.ExportedType1{}.Method1()}

	t2 := struct {
		s string
	}{
		s: other_pkg.ExportedType2{}.Method1(),
	}

	_, _ = t1, t2
}

func InSingleReturnAsValue() func() string {
	return other_pkg.ExportedType1{}.Method1
}

func InMultiReturnAsValue() (func() string, func() string) {
	return other_pkg.ExportedType1{}.Method1, other_pkg.ExportedType2{}.Method1
}

func InSingleReturnForReturn() func() string {
	return other_pkg.ExportedType1{}.Method1
}

func InMultiReturnForReturn() (string, string) {
	return other_pkg.ExportedType1{}.Method1(), other_pkg.ExportedType2{}.Method1()
}

func InDefer() {
	defer other_pkg.ExportedType1{}.Method1()
	t2 := other_pkg.ExportedType2{}
	defer t2.Method1()
}

func InGoroutine() {
	go other_pkg.ExportedType1{}.Method1()
	t2 := other_pkg.ExportedType2{}
	go t2.Method1()
}

func UseStructWithBothReceiverTypes() {
	s := &other_pkg.StructWithBothReceiverTypes{}
	s.PointerReceiver()
	s.ValueReceiver()
}

func UseStructWithAnonEmbeddedValue() {
	s := other_pkg.StructWithAnonEmbeddedValue{StructWithBothReceiverTypes: other_pkg.StructWithBothReceiverTypes{}}
	s.ValueReceiver()
}

func UseStructWithAnonEmbeddedPointer() {
	s := other_pkg.StructWithAnonEmbeddedPointer{StructWithBothReceiverTypes: &other_pkg.StructWithBothReceiverTypes{}}
	s.PointerReceiver()
	s.ValueReceiver()
}

func UseStructWithNamedEmbeddedValue() {
	s := other_pkg.StructWithNamedEmbeddedValue{Embed: other_pkg.StructWithBothReceiverTypes{}}
	s.Embed.ValueReceiver()
}

func UseStructWithNamedEmbeddedPointer() {
	s := other_pkg.StructWithNamedEmbeddedPointer{Embed: &other_pkg.StructWithBothReceiverTypes{}}
	s.Embed.PointerReceiver()
	s.Embed.ValueReceiver()
}
