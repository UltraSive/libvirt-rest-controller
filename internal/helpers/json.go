package helpers

import (
	"encoding/json"
)

// ToJson converts the interface to JSON
func ToJson(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}
