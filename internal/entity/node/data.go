package node

type DataType string

const (
	Struct    DataType = "struct"
	Class     DataType = "class"
	Interface DataType = "interface"
	Enum      DataType = "enum"
	Alias     DataType = "alias"
	Union     DataType = "union"
)

type FieldSlot struct {
	Name string
	URN  URN
}

type MethodSlot struct {
	Name string
	URN  URN
}

type Data struct {
	Node
	Location Location
	Field    []FieldSlot
	Methods  []MethodSlot
}
