package form

type Form struct {
	Title  string      `json:",omitempty"`
	Fields []Field     `json:"Fields"`
	New    interface{} `json:",omitempty"`
}
type Field struct {
	Name        string                 `json:",omitempty"`
	Title       string                 `json:",omitempty"`
	Type        string                 `json:",omitempty"`
	Placeholder string                 `json:",omitempty"`
	Required    bool                   `json:",omitempty"`
	Readonly    bool                   `json:",omitempty"`
	Pattern     string                 `json:",omitempty"`
	Minlength   int                    `json:",omitempty"`
	Maxlength   int                    `json:",omitempty"`
	Min         string                 `json:",omitempty"`
	Max         string                 `json:",omitempty"`
	Ops         map[string]interface{} `json:",omitempty"`
}
