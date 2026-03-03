package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// Formatter defines the interface for output formatting.
type Formatter interface {
	Print(data interface{}) error
}

// JSONFormatter outputs data as formatted JSON.
type JSONFormatter struct{}

func (f *JSONFormatter) Print(data interface{}) error {
	b, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(b))
	return nil
}
