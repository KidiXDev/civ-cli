package output

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
)

// TableFormatter outputs data as an ASCII table.
type TableFormatter struct {
	Headers []string
	RowFunc func(item interface{}) []string
}

// Print expects data to be a slice of items.
func (f *TableFormatter) Print(data interface{}) error {
	// A robust implementation would use reflection to slice the data.
	// For simplicity, we assume lists are passed as []interface{}
	items, ok := data.([]interface{})
	if !ok {
		return fmt.Errorf("table formatting requires []interface{}, got %T", data)
	}

	table := tablewriter.NewTable(os.Stdout, tablewriter.WithHeader(f.Headers))

	for _, item := range items {
		rowStr := f.RowFunc(item)
		rowAny := make([]any, len(rowStr))
		for i, v := range rowStr {
			rowAny[i] = v
		}
		table.Append(rowAny...)
	}
	table.Render()
	return nil
}
