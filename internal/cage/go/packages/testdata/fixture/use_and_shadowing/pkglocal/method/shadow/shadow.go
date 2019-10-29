package shadow

const ExportedConst1 = ""
const ExportedConst2 = ""

type typeWithShadowReceiver struct{}

func (ExportedConst1 typeWithShadowReceiver) Method1() {
	ExportedConst2 := 2
	_ = ExportedConst2
}

type typeWithShadowParam struct{}

func (s typeWithShadowParam) Method1(ExportedConst1 string) {
	ExportedConst2 := 2
	_ = ExportedConst2
}
