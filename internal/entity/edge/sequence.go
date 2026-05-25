package edge

type SequenceType string

const (
	Situational SequenceType = "situational"
	Basic       SequenceType = "basic"
)

type Sequence struct {
	Edge
	Order int          `json:"order"`
	Type  SequenceType `json:"type"`
}
