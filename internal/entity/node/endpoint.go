package node

type Request struct {
	QueryParam Data
	Header     Data
	Body       Data
}

type Endpoint struct {
	Node
	Service     URN
	Module      URN
	Method      URN
	Route       string
	Hanlder     URN
	Request     Request
	Response    Data
	FeatureTags []string
}
