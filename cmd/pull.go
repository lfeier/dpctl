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
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/lfeier/dpctl/log"
	"github.com/lfeier/dpctl/util"
	"github.com/spf13/cobra"
)

func init() {
	var scmd = &cobra.Command{
		Use:     "pull",
		Short:   "Pull DataPower configuration objects and files",
		Long:    ``,
		PreRunE: preRunPull,
		RunE:    runPull,
	}

	CmdRoot.AddCommand(scmd)

	addVerboseFlag(scmd)
	addDPRestMgmtURLFlag(scmd)
	addDPUserNameFlag(scmd)
	addDPUserPasswordFlag(scmd)
	addDomainFlag(scmd)
	addHTTPTimeoutFlag(scmd)
	addProjectDirFlag(scmd)
	addPkgTagsFlag(scmd)
	addObjectsFlag(scmd)
	addFilesFlag(scmd)
	addIgnoreObjectsFlag(scmd)
	addIgnoreFilesFlag(scmd)
}

func preRunPull(cmd *cobra.Command, args []string) error {
	level, _ := getVerboseFlagValue(cmd)
	log.SetVebosity(level)

	return nil
}

func runPull(cmd *cobra.Command, args []string) error {
	dpRestMgmtURL, _ := getDPRestMgmtURLFlagValue(cmd)
	log.DbgLogger2.Printf("--dp-rest-mgmt-url=%v", dpRestMgmtURL)

	dpUserName, _ := getDPUserNameFlagValue(cmd)
	log.DbgLogger2.Printf("--dp-user-name=%v", dpUserName)

	dpUserPassword, _ := getDPUserPasswordFlagValue(cmd)
	log.DbgLogger2.Printf("--dp-user-password=%v", "********")

	domain, _ := getDomainFlagValue(cmd)
	log.DbgLogger2.Printf("--domain=%v", domain)

	httpTimeout, _ := getHTTPTimeoutFlagValue(cmd)
	log.DbgLogger2.Printf("--http-timeout=%v", httpTimeout)

	projectDir, _ := getProjectDirFlagValue(cmd)
	log.DbgLogger2.Printf("--project-dir=%v", projectDir)

	pkgTags, _ := getPkgTagsValue(cmd)
	log.DbgLogger2.Printf("--pkg-tags=%v", pkgTags)

	objects, _ := getObjectsFlagValue(cmd)
	log.DbgLogger2.Printf("--objects=%v", objects)

	files, _ := getFilesFlagValue(cmd)
	log.DbgLogger2.Printf("--files=%v", files)

	ignoreObjects, _ := getIgnoreObjectsFlagValue(cmd)
	log.DbgLogger2.Printf("--ignore-objects=%v", ignoreObjects)

	ignoreFiles, _ := getIgnoreFilesFlagValue(cmd)
	log.DbgLogger2.Printf("--ignore-files=%v", ignoreFiles)

	reObjects := regexp.MustCompile(strings.Join(objects, "|"))
	log.DbgLogger3.Println("objects regexp:", reObjects.String())

	reFiles := regexp.MustCompile(strings.Join(files, "|"))
	log.DbgLogger3.Println("files regexp:", reFiles.String())

	reIgnoreObjects := regexp.MustCompile(strings.Join(ignoreObjects, "|"))
	log.DbgLogger3.Println("ignore objects regexp:", reIgnoreObjects.String())

	reIgnoreFiles := regexp.MustCompile(strings.Join(ignoreFiles, "|"))
	log.DbgLogger3.Println("ignore files regexp:", reIgnoreFiles.String())

	allPackages, err := util.ProjectPackages(projectDir)
	if err != nil {
		return err
	}

	log.DbgLogger3.Println("all project packages:")
	for _, pkg := range allPackages {
		log.DbgLogger3.Println("  ", *pkg)
	}

	pkgs := util.FilterPackages(allPackages, pkgTags)
	if len(pkgs) == 0 {
		return errors.New("no packages selected")
	}

	log.DbgLogger3.Println("project packages selected:")
	for _, pkg := range pkgs {
		log.DbgLogger3.Println("  ", *pkg)
	}

	httpClient := util.CreateHTTPClient(httpTimeout)

	if err := pullObjects(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, reObjects, reIgnoreObjects, pkgs); err != nil {
		return err
	}

	if err := pullFiles(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, reFiles, reIgnoreFiles, pkgs); err != nil {
		return err
	}

	return nil
}

func pullObjects(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain string, reObjects, reIgnoreObjects *regexp.Regexp, pkgs util.PackageSlice) error {
	classes, err := util.GetObjectClasses(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword)
	if err != nil {
		return err
	}

	for _, cls := range classes {
		objects, err := util.GetObjects(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, cls)
		if err != nil {
			return err
		}

		for _, obj := range objects {
			qn := util.ObjectQName(cls, obj)

			if !reObjects.MatchString(qn) || reIgnoreObjects.MatchString(qn) {
				log.DbgLogger1.Println("Object ignored:", qn)
				continue
			}

			deleteLinks(obj.(util.GenericMap))

			pkg, err := util.GetObjectPackage(pkgs, qn)
			if err != nil {
				return err
			}

			if pkg == nil {
				pkg = pkgs[0]
			}

			f, err := util.SaveObject(pkg.Dir, cls, obj)
			if err != nil {
				return err
			}

			log.OutLogger.Println("Object saved:", f)
		}
	}

	return nil
}

func pullFiles(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain string, reFiles, reIgnoreFiles *regexp.Regexp, pkgs util.PackageSlice) error {
	stores, err := util.GetFileStores(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain)
	if err != nil {
		return err
	}

	walkDir := func(path string) error {
		if reIgnoreFiles.MatchString(path) || reIgnoreFiles.MatchString(fmt.Sprintf("%s/", path)) {
			log.DbgLogger1.Println("Directory ignored:", path)
			return util.ErrSkipDir
		}

		return nil
	}

	walkFile := func(path string, modified string, size uint) error {
		if !reFiles.MatchString(path) || reIgnoreFiles.MatchString(path) {
			log.DbgLogger1.Println("File ignored:", path)
			return nil
		}

		data, err := util.GetFile(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, path)
		if err != nil {
			log.ErrLogger.Println("Failed to retrive file:", path, err)
			return nil
		}

		pkg, err := util.GetFilePackage(pkgs, path)
		if err != nil {
			return err
		}

		if pkg == nil {
			pkg = pkgs[0]
		}

		f, err := util.SaveFile(pkg.Dir, path, data)
		if err != nil {
			return err
		}

		log.OutLogger.Println("File saved:", f)

		return nil
	}

	for _, store := range stores {
		if store == "cert" || store == "sharedcert" || store == "pubcert" {
			log.DbgLogger1.Println("Store ignored:", store)
			continue
		}

		err = util.WalkFileStore(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, store, walkDir, walkFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func deleteLinks(o util.GenericMap) {
	for k, v := range o {
		switch k {
		case "_links":
			delete(o, k)
			continue
		case "href":
			delete(o, k)
			continue
		}

		if reflect.TypeOf(v).Kind() == reflect.Map {
			deleteLinks(v.(util.GenericMap))
		}
	}
}
