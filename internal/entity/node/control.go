package node

type ControlType string

const (
	If     ControlType = "if"     // at least two edges, one for true and one for false (what about the no condition case?)
	Switch ControlType = "switch" // many edges, each with a condition (what about the default case?)
	Try    ControlType = "try"    //
	Catch  ControlType = "catch"
	While  ControlType = "while" // one conditional. at least two edges, one for the condition being true and one for the condition being false (what about the no condition case?)
	For    ControlType = "for"   // one conditional. at least two edges, one for the condition being true and one for the condition being false (what about the no condition case?)
)

type Control struct {
	Node
	Type ControlType `json:"type"`
}
