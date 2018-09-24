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

package cmd

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestPullCmdFlags(t *testing.T) {
	a := []string{
		"pull",
	}
	cmd, _, err := CmdRoot.Find(a)
	if err != nil {
		t.Fatal(err)
	}

	n := 0
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		switch f.Name {
		case
			"verbose",
			"dp-rest-mgmt-url",
			"dp-user-name",
			"dp-user-password",
			"domain",
			"http-timeout",
			"project-dir",
			"pkg-tags",
			"objects",
			"files",
			"ignore-objects",
			"ignore-files":
			n++
		default:
			t.Errorf("Unknown flag '%v'", f.Name)
		}
	})

	expected := 12
	if n != expected {
		t.Errorf("Expected '%v' flags, got '%v'", expected, n)
	}
}
