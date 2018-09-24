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
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestCmdFlags(t *testing.T) {
	cmd := &cobra.Command{}

	addDPRestMgmtURLFlag(cmd)
	addDPUserNameFlag(cmd)
	addDPUserPasswordFlag(cmd)
	addDomainFlag(cmd)
	addHTTPTimeoutFlag(cmd)
	addProjectDirFlag(cmd)
	addObjectsFlag(cmd)
	addFilesFlag(cmd)
	addIgnoreObjectsFlag(cmd)
	addIgnoreFilesFlag(cmd)

	err := cmd.Flags().Set("dp-rest-mgmt-url", "str1")
	if err != nil {
		t.Error(err)
	}

	err = cmd.Flags().Set("dp-user-name", "str2")
	if err != nil {
		t.Error(err)
	}

	err = cmd.Flags().Set("dp-user-password", "str3")
	if err != nil {
		t.Error(err)
	}

	err = cmd.Flags().Set("domain", "str4")
	if err != nil {
		t.Error(err)
	}

	err = cmd.Flags().Set("http-timeout", "8s")
	if err != nil {
		t.Error(err)
	}

	err = cmd.Flags().Set("objects", "str6.1")
	if err != nil {
		t.Error(err)
	}
	err = cmd.Flags().Set("objects", "str6.2")
	if err != nil {
		t.Error(err)
	}

	err = cmd.Flags().Set("files", "str7.1")
	if err != nil {
		t.Error(err)
	}
	err = cmd.Flags().Set("files", "str7.2")
	if err != nil {
		t.Error(err)
	}

	err = cmd.Flags().Set("ignore-objects", "str8.1")
	if err != nil {
		t.Error(err)
	}
	err = cmd.Flags().Set("ignore-objects", "str8.2")
	if err != nil {
		t.Error(err)
	}

	err = cmd.Flags().Set("ignore-files", "str9.1")
	if err != nil {
		t.Error(err)
	}
	err = cmd.Flags().Set("ignore-files", "str9.2")
	if err != nil {
		t.Error(err)
	}

	v1, err := getDPRestMgmtURLFlagValue(cmd)
	if err != nil {
		t.Error(err)
	}
	if v1 != "str1" {
		t.Errorf("Expected '%v', got '%v'", "str1", v1)
	}

	v2, err := getDPUserNameFlagValue(cmd)
	if err != nil {
		t.Error(err)
	}
	if v2 != "str2" {
		t.Errorf("Expected '%v', got '%v'", "str2", v2)
	}

	v3, err := getDPUserPasswordFlagValue(cmd)
	if err != nil {
		t.Error(err)
	}
	if v3 != "str3" {
		t.Errorf("Expected '%v', got '%v'", "str3", v3)
	}

	v4, err := getDomainFlagValue(cmd)
	if err != nil {
		t.Error(err)
	}
	if v4 != "str4" {
		t.Errorf("Expected '%v', got '%v'", "str4", v4)
	}

	v5, err := getHTTPTimeoutFlagValue(cmd)
	if err != nil {
		t.Error(err)
	}
	if v5.Seconds() != 8 {
		t.Errorf("Expected '%v', got '%v'", "8s", v5)
	}

	v6, err := getObjectsFlagValue(cmd)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(v6, []string{"str6.1", "str6.2"}) {
		t.Errorf("Expected '%v', got '%v'", []string{"str6.1", "str6.2"}, v6)
	}

	v7, err := getFilesFlagValue(cmd)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(v7, []string{"str7.1", "str7.2"}) {
		t.Errorf("Expected '%v', got '%v'", []string{"str7.1", "str7.2"}, v7)
	}

	v8, err := getIgnoreObjectsFlagValue(cmd)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(v8, []string{"str8.1", "str8.2"}) {
		t.Errorf("Expected '%v', got '%v'", []string{"str8.1", "str8.2"}, v8)
	}

	v9, err := getIgnoreFilesFlagValue(cmd)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(v9, []string{"str9.1", "str9.2"}) {
		t.Errorf("Expected '%v', got '%v'", []string{"str9.1", "str9.2"}, v9)
	}
}
