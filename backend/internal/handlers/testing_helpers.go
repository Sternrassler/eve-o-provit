// Package handlers - Test helper functions
package handlers

import (
	"encoding/json"
	"io"
)

// parseJSON is a test helper to parse JSON response body
func parseJSON(body io.Reader, dest interface{}) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}
