package node

type Location struct {
	File     string `json:"file,omitempty"`
	LineInit int    `json:"line_init,omitempty"`
	LineEnd  int    `json:"line_end,omitempty"`
	ColInit  int    `json:"col_init,omitempty"`
	ColEnd   int    `json:"col_end,omitempty"`
}
