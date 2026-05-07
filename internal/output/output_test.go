package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrinterJSON(t *testing.T) {
	var out bytes.Buffer
	p := New(JSON, &out, &bytes.Buffer{})

	if err := p.JSON(map[string]string{"hello": "world"}); err != nil {
		t.Fatalf("JSON() error = %v", err)
	}

	if !strings.Contains(out.String(), "\"hello\": \"world\"") {
		t.Fatalf("JSON output = %q", out.String())
	}
}

func TestPrinterLinesAndErrorf(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	p := New(Human, &out, &errOut)

	p.Lines([]string{"a", "b"})
	p.Errorf("oops %d", 2)

	if got := out.String(); got != "a\nb\n" {
		t.Fatalf("Lines output = %q", got)
	}
	if got := errOut.String(); got != "oops 2\n" {
		t.Fatalf("Errorf output = %q", got)
	}
}

func TestPrinterTableHumanAndPlain(t *testing.T) {
	rows := [][]string{{"1", "two"}}

	var humanOut bytes.Buffer
	if err := New(Human, &humanOut, &bytes.Buffer{}).Table([]string{"A", "B"}, rows); err != nil {
		t.Fatalf("Human Table() error = %v", err)
	}
	if !strings.Contains(humanOut.String(), "A") {
		t.Fatalf("human table output = %q", humanOut.String())
	}

	var plainOut bytes.Buffer
	if err := New(Plain, &plainOut, &bytes.Buffer{}).Table([]string{"A", "B"}, rows); err != nil {
		t.Fatalf("Plain Table() error = %v", err)
	}
	if strings.Contains(plainOut.String(), "A") {
		t.Fatalf("plain table should not include headers: %q", plainOut.String())
	}

	if New(Plain, &plainOut, &bytes.Buffer{}).Mode() != Plain {
		t.Fatal("Mode() unexpected")
	}
}
