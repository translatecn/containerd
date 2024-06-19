/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package bandwidth

import (
	"regexp"
)

var (
	classShowMatcher      = regexp.MustCompile(`class htb (1:\d+)`)
	classAndHandleMatcher = regexp.MustCompile(`filter parent 1:.*fh (\d+::\d+).*flowid (\d+:\d+)`)
)

// NewTCShaper makes a new tcShaper for the given interface

// Convert a CIDR from text to a hex representation
// Strips any masked parts of the IP, so 1.2.3.4/16 becomes hex(1.2.0.0)/ffffffff

// Convert a CIDR from hex representation to text, opposite of the above.
