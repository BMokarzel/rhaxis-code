package node

type FunctionParam struct {
	Name string `json:"name"`
	Type Data   `json:"type"`
}

type Function struct {
	Node
	Service  URN
	Module   URN
	Class    URN
	Param    []FunctionParam
	Return   []Data
	Location Location
}

func (f *Function) isMethod() bool {
	return f.Class != URN("")
}
