package xjson

import (
	"encoding/json"
	"fmt"
	"io"
)

func Decode[T any](r io.Reader) (T, error) {
	var output T
	if err := json.NewDecoder(r).Decode(&output); err != nil {
		return *new(T), fmt.Errorf("error decoding json: %w", err)
	}
	return output, nil
}

func MustDecode[T any](r io.Reader) T {
	v, err := Decode[T](r)
	if err != nil {
		panic(err)
	}
	return v
}
