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

import "fmt"

// GenericMap is used to store arbitrary JSON objects
type GenericMap = map[string]interface{}

// GenericArray is used to store arbitrary JSON arrays
type GenericArray = []interface{}

// JSONValue returns a value at a specific path
func JSONValue(jsonData interface{}, p ...interface{}) interface{} {
	c := jsonData
	for _, v := range p {
		switch t := v.(type) {
		case string:
			if m, ok := c.(GenericMap); ok {
				c = m[v.(string)]
			} else {
				return nil
			}
		case int:
			if a, ok := c.(GenericArray); ok {
				c = a[v.(int)]
			} else {
				return nil
			}
		default:
			panic(fmt.Sprintf("unknown json path type: %v", t))
		}
	}

	return c
}
