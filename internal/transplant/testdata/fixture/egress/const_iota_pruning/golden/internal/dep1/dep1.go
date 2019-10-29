package dep1

const (
	OnlyUsedInIota1 = 1 // should remain because E1 depends on it
	OnlyUsedInIota2 = 2 // should remain because F1 depends on it
	OnlyUsedInIota3 = 3 // should remain because I1 depends on it
)

const (
	A1 = iota                   // shouild remain due to iota (even though not used)
	A2                          // shouild remain due to iota (even though not used)
	C1 = iota                   // should remtain due to iota (even though not used)
	C2                          // shouild remain due to iota (even though not used)
	E1 = iota + OnlyUsedInIota1 // shouild remain due to iota (even though not used)
	E2                          // shouild remain due to iota (even though not used)
	F1 = iota + OnlyUsedInIota2 // shouild remain due to iota (even though not used)
	F2                          // shouild remain due to iota (even though not used)
	G1 = ""                     // should remain due to use
	H1 = iota                   // shouild remain due to iota (though also used)
	I1 = iota + OnlyUsedInIota3 // shouild remain due to iota (though also used)
)
