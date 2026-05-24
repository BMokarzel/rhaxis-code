package node

type HttpMethod struct {
	Node
}

var MethodGet HttpMethod = HttpMethod{
	Node: Node{
		URN:  "",
		Type: HttpMethodNode,
		Meta: Meta{},
	},
}

var MethodPost HttpMethod = HttpMethod{
	Node: Node{
		URN:  "",
		Type: HttpMethodNode,
		Meta: Meta{},
	},
}

var MethodPut HttpMethod = HttpMethod{
	Node: Node{
		URN:  "",
		Type: HttpMethodNode,
		Meta: Meta{},
	},
}

var MethodPatch HttpMethod = HttpMethod{
	Node: Node{
		URN:  "",
		Type: HttpMethodNode,
		Meta: Meta{},
	},
}

var MethodDelete HttpMethod = HttpMethod{
	Node: Node{
		URN:  "",
		Type: HttpMethodNode,
		Meta: Meta{},
	},
}
