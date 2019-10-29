package use

func nonExportedFunc1() string { return "" }
func ExportedFunc1() string    { return "" }
func ExportedFunc2() string    { return "" }
func ExportedFunc3() string    { return "" }
func ExportedFunc4() chan int  { return make(chan int) }

func init() {
	a := nonExportedFunc1
	b := ExportedFunc1()
	_, _ = a, b
}

func InCallAsValue() {
	f := func(v func() string) {}
	f(nonExportedFunc1)
}

func InCallForReturn() {
	f := func(v string) {}
	f(ExportedFunc1())
}

func InIfForReturn() {
	if nonExportedFunc1() == "" {
	}
}

func InSwitchForReturn() {
	switch ExportedFunc1() {
	}
}

func InSwitchCaseForReturn() {
	var a string
	switch a {
	case nonExportedFunc1():
	}
}

func InSelectCaseForReturn() {
	select {
	case <-ExportedFunc4():
	}
}

func InSelectCaseAssignForReturn() {
	select {
	case f := <-ExportedFunc4():
		_ = f
	}
}

func InAssignAsValue() {
	var a interface{} = ExportedFunc1
	_ = a
}

func InAssignShortAsValue() {
	a := nonExportedFunc1
	b, c := ExportedFunc1, ExportedFunc3
	_, _, _ = a, b, c
}

func InAssignForReturn() {
	var a string = nonExportedFunc1()
	_ = a
}

func InAssignShortForReturn() {
	a := nonExportedFunc1()
	b, c := ExportedFunc1(), ExportedFunc3()
	_, _, _ = a, b, c
}

func InCompositeLitAsValue() {
	type T struct {
		f func() string
	}
	t1 := T{f: nonExportedFunc1}

	t2 := struct {
		f func() string
	}{
		f: ExportedFunc1,
	}

	_, _ = t1, t2
}

func InCompositeLitForReturn() {
	type T struct {
		s string
	}
	t1 := T{s: nonExportedFunc1()}

	t2 := struct {
		s string
	}{
		s: ExportedFunc1(),
	}

	_, _ = t1, t2
}

func InSingleReturnAsValue() func() string {
	return ExportedFunc1
}

func InMultiReturnAsValue() (func() string, func() string) {
	return nonExportedFunc1, ExportedFunc1
}

func InSingleReturnForReturn() func() string {
	return ExportedFunc1
}

func InMultiReturnForReturn() (string, string) {
	return nonExportedFunc1(), ExportedFunc1()
}

func InDefer() {
	defer ExportedFunc1()
}

func InGoroutine() {
	go ExportedFunc1()
}
