package output

import (
	"encoding/json"
	"fmt"
	"io"
)

type Printer struct {
	Mode string
	Out  io.Writer
}

func New(mode string, out io.Writer) *Printer {
	if mode == "" {
		mode = "human"
	}
	return &Printer{Mode: mode, Out: out}
}

func (p *Printer) JSONEnabled() bool {
	return p.Mode == "json"
}

func (p *Printer) Print(data any, human func(io.Writer) error) error {
	if p.JSONEnabled() {
		enc := json.NewEncoder(p.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}
	return human(p.Out)
}

func KeyValue(w io.Writer, key, value string) error {
	_, err := fmt.Fprintf(w, "%-14s %s\n", key+":", value)
	return err
}
