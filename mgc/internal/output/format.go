// Package output provides output formatting for mgc commands.
// Supported formats: json, table, yaml, tsv.
// Supports JMESPath query filtering via the --query flag.
package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/jmespath/go-jmespath"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v3"
)

// Printer writes formatted output to a writer.
type Printer struct {
	Out    io.Writer
	ErrOut io.Writer
}

// New creates a Printer writing to stdout/stderr.
func New() *Printer {
	return &Printer{Out: os.Stdout, ErrOut: os.Stderr}
}

// Print formats and prints data in the given format after applying an optional JMESPath query.
// data must be JSON-serializable.
func (p *Printer) Print(data interface{}, format string, query string) error {
	// Convert to a JSON-friendly representation first
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	var parsed interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		return fmt.Errorf("failed to parse data: %w", err)
	}

	// Apply JMESPath filter if provided
	if query != "" {
		result, err := jmespath.Search(query, parsed)
		if err != nil {
			return fmt.Errorf("invalid JMESPath query %q: %w", query, err)
		}
		parsed = result
	}

	switch strings.ToLower(format) {
	case "json", "":
		return p.printJSON(parsed)
	case "table":
		return p.printTable(parsed)
	case "yaml":
		return p.printYAML(parsed)
	case "tsv":
		return p.printTSV(parsed)
	default:
		return fmt.Errorf("unsupported output format %q: use json, table, yaml, or tsv", format)
	}
}

func (p *Printer) printJSON(data interface{}) error {
	enc := json.NewEncoder(p.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func (p *Printer) printYAML(data interface{}) error {
	return yaml.NewEncoder(p.Out).Encode(data)
}

func (p *Printer) printTable(data interface{}) error {
	// Handle scalar values — just print them
	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Slice {
		// Single object or scalar: render as key-value table
		return p.printObjectAsTable(data)
	}

	// Slice: render as multi-row table
	slice, ok := data.([]interface{})
	if !ok || len(slice) == 0 {
		fmt.Fprintln(p.Out, "(no results)")
		return nil
	}

	// Collect headers from first object's keys
	headers := getObjectKeys(slice[0])

	table := tablewriter.NewWriter(p.Out)
	table.SetHeader(headers)
	table.SetAutoWrapText(false)
	table.SetBorder(true)

	for _, item := range slice {
		row := make([]string, len(headers))
		obj, ok := item.(map[string]interface{})
		if !ok {
			row[0] = fmt.Sprintf("%v", item)
		} else {
			for i, h := range headers {
				row[i] = valueToString(obj[h])
			}
		}
		table.Append(row)
	}

	table.Render()
	return nil
}

func (p *Printer) printObjectAsTable(data interface{}) error {
	obj, ok := data.(map[string]interface{})
	if !ok {
		fmt.Fprintln(p.Out, valueToString(data))
		return nil
	}

	table := tablewriter.NewWriter(p.Out)
	table.SetHeader([]string{"KEY", "VALUE"})
	table.SetAutoWrapText(true)
	table.SetBorder(true)

	// Sort keys for consistent output
	keys := getObjectKeys(data)
	for _, k := range keys {
		table.Append([]string{k, valueToString(obj[k])})
	}

	table.Render()
	return nil
}

func (p *Printer) printTSV(data interface{}) error {
	var buf bytes.Buffer

	slice, ok := data.([]interface{})
	if !ok {
		// Single value
		fmt.Fprintf(&buf, "%s\n", valueToString(data))
		_, err := io.Copy(p.Out, &buf)
		return err
	}

	if len(slice) == 0 {
		return nil
	}

	headers := getObjectKeys(slice[0])
	// Print header row
	fmt.Fprintln(&buf, strings.Join(headers, "\t"))

	for _, item := range slice {
		obj, ok := item.(map[string]interface{})
		if !ok {
			fmt.Fprintf(&buf, "%s\n", valueToString(item))
			continue
		}
		row := make([]string, len(headers))
		for i, h := range headers {
			row[i] = valueToString(obj[h])
		}
		fmt.Fprintln(&buf, strings.Join(row, "\t"))
	}

	_, err := io.Copy(p.Out, &buf)
	return err
}

// getObjectKeys returns sorted keys of a JSON object.
func getObjectKeys(v interface{}) []string {
	obj, ok := v.(map[string]interface{})
	if !ok {
		return []string{"value"}
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// valueToString converts any value to a display string.
func valueToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case map[string]interface{}:
		b, _ := json.Marshal(val)
		return string(b)
	case []interface{}:
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = valueToString(item)
		}
		return strings.Join(parts, ", ")
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}
