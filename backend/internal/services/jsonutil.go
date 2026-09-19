package services

import (
	"encoding/json"
	"io"
)

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }

func jsonDecode(r io.Reader, dest any) error {
	return json.NewDecoder(r).Decode(dest)
}
