package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

type Mode int

const (
	Human Mode = iota
	JSON
	Plain
)

type Printer struct {
	mode   Mode
	out    io.Writer
	errOut io.Writer
}

func New(mode Mode, out io.Writer, errOut io.Writer) *Printer {
	return &Printer{mode: mode, out: out, errOut: errOut}
}

func (p *Printer) Mode() Mode {
	return p.mode
}

func (p *Printer) JSON(v any) error {
	encoder := json.NewEncoder(p.out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func (p *Printer) Lines(lines []string) {
	for _, line := range lines {
		_, _ = fmt.Fprintln(p.out, line)
	}
}

func (p *Printer) Table(headers []string, rows [][]string) error {
	tw := tabwriter.NewWriter(p.out, 0, 0, 2, ' ', 0)
	if p.mode == Human && len(headers) > 0 {
		if _, err := fmt.Fprintln(tw, joinTabRow(headers)); err != nil {
			return err
		}
	}

	for _, row := range rows {
		if _, err := fmt.Fprintln(tw, joinTabRow(row)); err != nil {
			return err
		}
	}

	return tw.Flush()
}

func (p *Printer) Errorf(format string, args ...any) {
	_, _ = fmt.Fprintf(p.errOut, format+"\n", args...)
}

func joinTabRow(values []string) string {
	row := ""
	for i, value := range values {
		if i > 0 {
			row += "\t"
		}
		row += value
	}
	return row
}
