package dto

type NodeDirectDependencie struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	Relashionship string `json:"relationship"`
}

type NodeInfo struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"`
	Name         string                  `json:"name"`
	Properties   map[string]any          `json:"properties,omitempty"`
	Metadata     map[string]string       `json:"metadata,omitempty"`
	Dependencies []NodeDirectDependencie `json:"dependencies,omitempty"`
}

// com outros nós completos dentro
type NodeResponse struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	Name         string            `json:"name"`
	Properties   map[string]any    `json:"properties,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Dependencies []NodeResponse    `json:"dependencies,omitempty"`
}

type NodeListResponse struct {
	Node []NodeResponse `json:"node"`
	Edge string
}

type ServiceNodes struct {
	Service []NodeInfo
}
