// Copyright 2015 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"fmt"
	"strings"
)

// UnmarshallableBool typedef for builtin bool
// because builtin type's methods can't be declared
type UnmarshallableBool bool

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// Returns boolean true if the string is "1" or "[Tt]rue"
// Returns boolean false if the string is "0" or "[Ff]alse"
func (b *UnmarshallableBool) UnmarshalText(data []byte) error {
	s := strings.ToLower(string(data))
	switch s {
	case "1", "true":
		*b = true
	case "0", "false":
		*b = false
	default:
		return fmt.Errorf("boolean unmarshal error: invalid input %s", s)
	}
	return nil
}

// UnmarshallableString typedef for builtin string
type UnmarshallableString string

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// Returns the string
func (s *UnmarshallableString) UnmarshalText(data []byte) error {
	*s = UnmarshallableString(data)
	return nil
}

// CommonArgs contains the IgnoreUnknown argument
// and must be embedded by all Arg structs
type CommonArgs struct {
	IgnoreUnknown UnmarshallableBool `json:"ignoreunknown,omitempty"`
}

// GetKeyField is a helper function to receive Values
// Values that represent a pointer to a struct

// UnmarshalableArgsError is used to indicate error unmarshalling args
// from the args-string in the form "K=V;K2=V2;..."
type UnmarshalableArgsError struct {
	error
}

// LoadArgs parses args from a string in the form "K=V;K2=V2;..."
