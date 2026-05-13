package config

import "encoding/json"

func marshalJSON(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func unmarshalJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
