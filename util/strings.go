// Copyright © 2018 Lucian Feier
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

package util

import (
	"regexp"
	"strings"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

// SpinalCase converts a string to spinal case
func SpinalCase(str string) string {
	tmp := matchFirstCap.ReplaceAllString(str, "${1}-${2}")
	tmp = matchAllCap.ReplaceAllString(tmp, "${1}-${2}")

	return strings.ToLower(tmp)
}
