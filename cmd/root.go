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
	"time"

	"github.com/spf13/cobra"
)

func init() {
}

// CmdRoot is the root command for the application
var CmdRoot = &cobra.Command{
	Use:   "dpctl",
	Short: "Root command",
	Long:  ``,
}

func addVerboseFlag(cmd *cobra.Command) {
	cmd.Flags().CountP("verbose", "v", "verbose mode")
}

func addDPRestMgmtURLFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("dp-rest-mgmt-url", "u", "", "DataPower REST management url")
}

func addDPUserNameFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("dp-user-name", "n", "", "DataPower user name")
}

func addDPUserPasswordFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("dp-user-password", "p", "", "DataPower user password")
}

func addDomainFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("domain", "d", "", "DataPower domain")
}

func addHTTPTimeoutFlag(cmd *cobra.Command) {
	cmd.Flags().Duration("http-timeout", time.Duration(600)*time.Second, "HTTP timeout")
}

func addProjectDirFlag(cmd *cobra.Command) {
	cmd.Flags().String("project-dir", "./", "prject directory")
}

func addPkgTagsFlag(cmd *cobra.Command) {
	cmd.Flags().StringSlice("pkg-tags", []string{}, "package selector")
}

func addObjectsFlag(cmd *cobra.Command) {
	cmd.Flags().StringSlice("objects", []string{".*"}, "objects regex filter")
}

func addFilesFlag(cmd *cobra.Command) {
	cmd.Flags().StringSlice("files", []string{".*"}, "files regex filter")
}

func addIgnoreObjectsFlag(cmd *cobra.Command) {
	cmd.Flags().StringSlice("ignore-objects", []string{"^.*/__.*__$"}, "ignore objects regex filter")
}

func addIgnoreFilesFlag(cmd *cobra.Command) {
	cmd.Flags().StringSlice("ignore-files", []string{"^(chkpoints/.*|config/.*|export/.*|image/.*|logstore/.*|logtemp/.*|policyframework/.*|pubcert/.*|sharedcert/.*|store/.*|tasktemplates/.*|temporary/.*)$"}, "ignore files regex filter")
}

func addParallelFlag(cmd *cobra.Command) {
	cmd.Flags().Int("parallel", 1, "allow parallel execution")
}

func getVerboseFlagValue(cmd *cobra.Command) (int, error) {
	return cmd.Flags().GetCount("verbose")
}

func getDPRestMgmtURLFlagValue(cmd *cobra.Command) (string, error) {
	return cmd.Flags().GetString("dp-rest-mgmt-url")
}

func getDPUserNameFlagValue(cmd *cobra.Command) (string, error) {
	return cmd.Flags().GetString("dp-user-name")
}

func getDPUserPasswordFlagValue(cmd *cobra.Command) (string, error) {
	return cmd.Flags().GetString("dp-user-password")
}

func getDomainFlagValue(cmd *cobra.Command) (string, error) {
	return cmd.Flags().GetString("domain")
}

func getHTTPTimeoutFlagValue(cmd *cobra.Command) (time.Duration, error) {
	return cmd.Flags().GetDuration("http-timeout")
}

func getProjectDirFlagValue(cmd *cobra.Command) (string, error) {
	return cmd.Flags().GetString("project-dir")
}

func getPkgTagsValue(cmd *cobra.Command) ([]string, error) {
	return cmd.Flags().GetStringSlice("pkg-tags")
}

func getObjectsFlagValue(cmd *cobra.Command) ([]string, error) {
	return cmd.Flags().GetStringSlice("objects")
}

func getFilesFlagValue(cmd *cobra.Command) ([]string, error) {
	return cmd.Flags().GetStringSlice("files")
}

func getIgnoreObjectsFlagValue(cmd *cobra.Command) ([]string, error) {
	return cmd.Flags().GetStringSlice("ignore-objects")
}

func getIgnoreFilesFlagValue(cmd *cobra.Command) ([]string, error) {
	return cmd.Flags().GetStringSlice("ignore-files")
}

func getParallelFlagValue(cmd *cobra.Command) (int, error) {
	return cmd.Flags().GetInt("parallel")
}
