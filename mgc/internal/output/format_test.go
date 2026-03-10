package output_test

import (
	"bytes"
	"testing"

	"github.com/chenxizhang/testagent/mgc/internal/output"
)

func TestPrintJSON(t *testing.T) {
	p := output.New()
	var buf bytes.Buffer
	p.Out = &buf

	data := []map[string]interface{}{
		{"id": "1", "name": "Alice"},
		{"id": "2", "name": "Bob"},
	}

	if err := p.Print(data, "json", ""); err != nil {
		t.Fatalf("Print JSON failed: %v", err)
	}

	got := buf.String()
	if got == "" {
		t.Error("expected non-empty JSON output")
	}
	if len(got) < 10 {
		t.Errorf("JSON output too short: %q", got)
	}
}

func TestPrintTable(t *testing.T) {
	p := output.New()
	var buf bytes.Buffer
	p.Out = &buf

	data := []map[string]interface{}{
		{"displayName": "Alice", "mail": "alice@contoso.com"},
	}

	if err := p.Print(data, "table", ""); err != nil {
		t.Fatalf("Print table failed: %v", err)
	}

	got := buf.String()
	if !bytes.Contains([]byte(got), []byte("Alice")) {
		t.Errorf("expected 'Alice' in table output, got: %s", got)
	}
}

func TestPrintYAML(t *testing.T) {
	p := output.New()
	var buf bytes.Buffer
	p.Out = &buf

	data := map[string]interface{}{"id": "123", "name": "Test"}

	if err := p.Print(data, "yaml", ""); err != nil {
		t.Fatalf("Print YAML failed: %v", err)
	}

	got := buf.String()
	if !bytes.Contains([]byte(got), []byte("id:")) {
		t.Errorf("expected 'id:' in YAML output, got: %s", got)
	}
}

func TestPrintTSV(t *testing.T) {
	p := output.New()
	var buf bytes.Buffer
	p.Out = &buf

	data := []map[string]interface{}{
		{"id": "1", "name": "Alice"},
	}

	if err := p.Print(data, "tsv", ""); err != nil {
		t.Fatalf("Print TSV failed: %v", err)
	}

	got := buf.String()
	if !bytes.Contains([]byte(got), []byte("\t")) {
		t.Errorf("expected tab in TSV output, got: %s", got)
	}
}

func TestPrintWithJMESPathQuery(t *testing.T) {
	p := output.New()
	var buf bytes.Buffer
	p.Out = &buf

	data := []map[string]interface{}{
		{"displayName": "Alice", "department": "Engineering"},
		{"displayName": "Bob", "department": "Sales"},
	}

	// Query to get just display names
	if err := p.Print(data, "json", "[].displayName"); err != nil {
		t.Fatalf("Print with JMESPath failed: %v", err)
	}

	got := buf.String()
	if !bytes.Contains([]byte(got), []byte("Alice")) {
		t.Errorf("expected 'Alice' in JMESPath filtered output, got: %s", got)
	}
	if bytes.Contains([]byte(got), []byte("Engineering")) {
		t.Errorf("expected 'Engineering' to be filtered out, got: %s", got)
	}
}

func TestPrintInvalidFormat(t *testing.T) {
	p := output.New()
	var buf bytes.Buffer
	p.Out = &buf

	if err := p.Print("data", "xml", ""); err == nil {
		t.Error("expected error for unsupported format 'xml'")
	}
}

func TestPrintEmptySlice(t *testing.T) {
	p := output.New()
	var buf bytes.Buffer
	p.Out = &buf

	if err := p.Print([]interface{}{}, "table", ""); err != nil {
		t.Fatalf("Print empty slice failed: %v", err)
	}

	got := buf.String()
	if !bytes.Contains([]byte(got), []byte("no results")) {
		t.Errorf("expected 'no results' for empty slice, got: %s", got)
	}
}

func TestPrintInvalidJMESPath(t *testing.T) {
	p := output.New()
	var buf bytes.Buffer
	p.Out = &buf

	data := []interface{}{"a", "b"}
	if err := p.Print(data, "json", "[[invalid"); err == nil {
		t.Error("expected error for invalid JMESPath")
	}
}
