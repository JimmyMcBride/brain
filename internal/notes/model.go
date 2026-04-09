package notes

type Note struct {
	ID       string         `json:"id"`
	Title    string         `json:"title"`
	Path     string         `json:"path"`
	Type     string         `json:"type"`
	Metadata map[string]any `json:"metadata"`
	Content  string         `json:"content"`
}
