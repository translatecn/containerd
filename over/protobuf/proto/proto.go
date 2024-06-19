// Package proto provides convinient aliases that make google.golang.org/protobuf migration easier.
package proto

import (
	google "google.golang.org/protobuf/proto"
)

func Marshal(input google.Message) ([]byte, error) {
	return google.Marshal(input)
}

func Unmarshal(input []byte, output google.Message) error {
	return google.Unmarshal(input, output)
}
