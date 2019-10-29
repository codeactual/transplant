package use

type nonExportedType1 struct{}

func (t nonExportedType1) Method1() string { return "" }

type ExportedType1 struct{}

func (t ExportedType1) Method1() string { return "" }

type ExportedType2 struct{}

func (t ExportedType2) Method1() string { return "" }

type ExportedType3 struct{}

func (t ExportedType3) Method1() string { return "" }

type ExportedType4 struct{}

func (t ExportedType4) Method1() chan int { return make(chan int) }

func init() {
	a := nonExportedType1{}.Method1
	b := ExportedType1{}.Method1()
	_, _ = a, b

	c := ExportedType3{}
	c.Method1()
}

func InCallAsValue() {
	f := func(v func() string) {}

	f(nonExportedType1{}.Method1)

	t1 := ExportedType1{}
	f(t1.Method1)
}

func InCallForReturn() {
	f := func(v string) {}

	f(nonExportedType1{}.Method1())

	t1 := ExportedType3{}
	f(t1.Method1())
}

func InIfForReturn() {
	if (nonExportedType1{}).Method1() == "" {
	}

	t1 := ExportedType1{}
	if t1.Method1() == "" {
	}
}

func InSwitchForReturn() {
	switch (nonExportedType1{}).Method1() {
	}

	t1 := ExportedType3{}
	switch t1.Method1() {
	}
}

func InSwitchCaseForReturn() {
	var a string
	switch a {
	case nonExportedType1{}.Method1():
	}

	var b string
	t1 := ExportedType1{}
	switch b {
	case t1.Method1():
	}
}

func InSelectCaseForReturn() {
	select {
	case <-ExportedType4{}.Method1():
	}
}

func InSelectCaseAssignForReturn() {
	select {
	case f := <-ExportedType4{}.Method1():
		_ = f
	}
}

func InAssignAsValue() {
	var a interface{} = nonExportedType1{}.Method1
	_ = a

	t1 := ExportedType3{}
	var b interface{} = t1.Method1
	_ = b
}

func InAssignShortAsValue() {
	a := ExportedType1{}.Method1
	b, c := ExportedType2{}.Method1, ExportedType3{}.Method1
	_, _, _ = a, b, c
}

func InAssignForReturn() {
	var a string = nonExportedType1{}.Method1()
	_ = a

	t1 := ExportedType1{}
	var b string = t1.Method1()
	_ = b
}

func InAssignShortForReturn() {
	a := ExportedType1{}.Method1()
	b, c := ExportedType2{}.Method1(), ExportedType3{}.Method1()
	_, _, _ = a, b, c
}

func InCompositeLitAsValue() {
	type T struct {
		f func() string
	}
	t1 := T{f: ExportedType1{}.Method1}

	t2 := struct {
		f func() string
	}{
		f: ExportedType2{}.Method1,
	}

	_, _ = t1, t2
}

func InCompositeLitForReturn() {
	type T struct {
		s string
	}
	t1 := T{s: ExportedType1{}.Method1()}

	t2 := struct {
		s string
	}{
		s: ExportedType2{}.Method1(),
	}

	_, _ = t1, t2
}

func InSingleReturnAsValue() func() string {
	return ExportedType1{}.Method1
}

func InMultiReturnAsValue() (func() string, func() string) {
	return ExportedType1{}.Method1, ExportedType2{}.Method1
}

func InSingleReturnForReturn() func() string {
	return ExportedType1{}.Method1
}

func InMultiReturnForReturn() (string, string) {
	return ExportedType1{}.Method1(), ExportedType2{}.Method1()
}

func InDefer() {
	defer ExportedType1{}.Method1()
	t2 := ExportedType2{}
	defer t2.Method1()
}

func InGoroutine() {
	go ExportedType1{}.Method1()
	t2 := ExportedType2{}
	go t2.Method1()
}

type StructWithBothReceiverTypes struct{}

func (s *StructWithBothReceiverTypes) PointerReceiver() {
}

func (s StructWithBothReceiverTypes) ValueReceiver() {
}

func UseStructWithBothReceiverTypes() {
	s := &StructWithBothReceiverTypes{}
	s.PointerReceiver()
	s.ValueReceiver()
}

type StructWithAnonEmbeddedValue struct {
	StructWithBothReceiverTypes
}

type StructWithAnonEmbeddedPointer struct {
	*StructWithBothReceiverTypes
}

func UseStructWithAnonEmbeddedValue() {
	s := StructWithAnonEmbeddedValue{StructWithBothReceiverTypes: StructWithBothReceiverTypes{}}
	s.ValueReceiver()
}

func UseStructWithAnonEmbeddedPointer() {
	s := StructWithAnonEmbeddedPointer{StructWithBothReceiverTypes: &StructWithBothReceiverTypes{}}
	s.PointerReceiver()
	s.ValueReceiver()
}

type StructWithNamedEmbeddedValue struct {
	Embed StructWithBothReceiverTypes
}

type StructWithNamedEmbeddedPointer struct {
	Embed *StructWithBothReceiverTypes
}

func UseStructWithNamedEmbeddedValue() {
	s := StructWithNamedEmbeddedValue{Embed: StructWithBothReceiverTypes{}}
	s.Embed.ValueReceiver()
}

func UseStructWithNamedEmbeddedPointer() {
	s := StructWithNamedEmbeddedPointer{Embed: &StructWithBothReceiverTypes{}}
	s.Embed.PointerReceiver()
	s.Embed.ValueReceiver()
}

type NoReceiverName struct{}

func (NoReceiverName) Method() {
}

func UseNoReceiverName() {
	t := NoReceiverName{}
	t.Method()
}
