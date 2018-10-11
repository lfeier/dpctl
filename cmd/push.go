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
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lfeier/dpctl/log"
	"github.com/lfeier/dpctl/util"
	"github.com/spf13/cobra"
	"golang.org/x/sync/semaphore"
)

func init() {
	var scmd = &cobra.Command{
		Use:    "push",
		Short:  "Push DataPower configuration objects and files",
		Long:   ``,
		PreRun: preRunPush,
		Run:    runPush,
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
	addParallelFlag(scmd)
}

func preRunPush(cmd *cobra.Command, args []string) {
	level, _ := getVerboseFlagValue(cmd)
	log.SetVebosity(level)
}

func runPush(cmd *cobra.Command, args []string) {
	if err := runPushE(cmd, args); err != nil {
		log.ErrLogger.Println("Error:", err.Error())
	}
}

func runPushE(cmd *cobra.Command, args []string) error {
	dpRestMgmtURL, _ := getDPRestMgmtURLFlagValue(cmd)
	log.DbgLogger1.Printf("--dp-rest-mgmt-url=%v", dpRestMgmtURL)

	dpUserName, _ := getDPUserNameFlagValue(cmd)
	log.DbgLogger1.Printf("--dp-user-name=%v", dpUserName)

	dpUserPassword, _ := getDPUserPasswordFlagValue(cmd)
	log.DbgLogger1.Printf("--dp-user-password=%v", "********")

	domain, _ := getDomainFlagValue(cmd)
	log.DbgLogger1.Printf("--domain=%v", domain)

	httpTimeout, _ := getHTTPTimeoutFlagValue(cmd)
	log.DbgLogger1.Printf("--http-timeout=%v", httpTimeout)

	projectDir, _ := getProjectDirFlagValue(cmd)
	log.DbgLogger1.Printf("--project-dir=%v", projectDir)

	pkgTags, _ := getPkgTagsValue(cmd)
	log.DbgLogger1.Printf("--pkg-tags=%v", pkgTags)

	objects, _ := getObjectsFlagValue(cmd)
	log.DbgLogger1.Printf("--objects=%v", objects)

	files, _ := getFilesFlagValue(cmd)
	log.DbgLogger1.Printf("--files=%v", files)

	ignoreObjects, _ := getIgnoreObjectsFlagValue(cmd)
	log.DbgLogger1.Printf("--ignore-objects=%v", ignoreObjects)

	ignoreFiles, _ := getIgnoreFilesFlagValue(cmd)
	log.DbgLogger1.Printf("--ignore-files=%v", ignoreFiles)

	parallel, _ := getParallelFlagValue(cmd)
	log.DbgLogger1.Printf("--parallel=%v", parallel)

	reObjects := regexp.MustCompile(strings.Join(objects, "|"))
	log.DbgLogger4.Println("objects regexp:", reObjects.String())

	reFiles := regexp.MustCompile(strings.Join(files, "|"))
	log.DbgLogger4.Println("files regexp:", reFiles.String())

	reIgnoreObjects := regexp.MustCompile(strings.Join(ignoreObjects, "|"))
	log.DbgLogger4.Println("ignore objects regexp:", reIgnoreObjects.String())

	reIgnoreFiles := regexp.MustCompile(strings.Join(ignoreFiles, "|"))
	log.DbgLogger4.Println("ignore files regexp:", reIgnoreFiles.String())

	allPackages, err := util.ProjectPackages(projectDir)
	if err != nil {
		return err
	}

	log.DbgLogger4.Println("all project packages:")
	for _, pkg := range allPackages {
		log.DbgLogger4.Println("  ", *pkg)
	}

	pkgs := util.FilterPackages(allPackages, pkgTags)
	if len(pkgs) == 0 {
		return errors.New("no packages selected")
	}

	log.DbgLogger1.Println("packages selected:")
	for _, pkg := range pkgs {
		log.DbgLogger1.Printf("  package: %s (priority %d)", pkg.Name, pkg.Priority)
	}

	httpClient := util.CreateHTTPClient(httpTimeout)

	sem := semaphore.NewWeighted(int64(parallel))

	err1 := pushFiles(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, reFiles, reIgnoreFiles, pkgs, sem, int64(parallel))

	err2 := pushObjects(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, reObjects, reIgnoreObjects, pkgs, sem, int64(parallel))

	if err1 != nil && err2 != nil {
		return fmt.Errorf("%s, %s", err1.Error(), err2.Error())
	}

	if err1 != nil {
		return err1
	}

	if err2 != nil {
		return err2
	}

	return nil
}

type pushResult int

const (
	pushError pushResult = iota
	pushOK
	pushNew
	pushSuccess
	pushDryRun
)

func (result *pushResult) String() string {
	names := [...]string{
		"ERROR",
		"OK",
		"NEW",
		"SUCCESS",
		"DRYRUN",
	}

	return names[*result]
}

var maxPushResultLength = 7

type logPushFile func(fileInfo *util.FileInfo, result *pushResult, start time.Time)
type logPushObject func(objectInfo *util.ObjectInfo, result *pushResult, start time.Time)

func pushFiles(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain string, reFiles, reIgnoreFiles *regexp.Regexp, pkgs util.PackageSlice, sem *semaphore.Weighted, n int64) error {
	files, err := util.GetProjectFiles(pkgs)
	if err != nil {
		return err
	}

	matchingFiles := files[:0]
	maxPathLength := 0
	for _, fileInfo := range files {
		if !reFiles.MatchString(fileInfo.Path) || reIgnoreFiles.MatchString(fileInfo.Path) {
			log.DbgLogger2.Println("file ignored:", fileInfo.Path)
			continue
		}

		matchingFiles = append(matchingFiles, fileInfo)
		if maxPathLength < len(fileInfo.Path) {
			maxPathLength = len(fileInfo.Path)
		}
	}

	maxPkgLength := 0
	for _, pkg := range pkgs {
		if maxPkgLength < len(pkg.Name) {
			maxPkgLength = len(pkg.Name)
		}
	}

	logFn := func(fileInfo *util.FileInfo, result *pushResult, start time.Time) {
		elapsed := time.Since(start)
		lf := fmt.Sprintf("FILE: %%-%ds [%%s] %%%ds [%%s]", maxPathLength, maxPkgLength+maxPushResultLength-len(fileInfo.Package.Name))
		log.OutLogger.Printf(lf, fileInfo.Path, fileInfo.Package.Name, result.String(), elapsed.Truncate(time.Millisecond).String())
	}

	ctx := context.TODO()
	var errCount uint64

	for _, fileInfo := range matchingFiles {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		go func(fileInfo *util.FileInfo) {
			defer sem.Release(1)
			if err := pushFile(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, fileInfo, logFn); err != nil {
				log.ErrLogger.Println("Error:", err.Error())
				atomic.AddUint64(&errCount, 1)
			}
		}(fileInfo)
	}

	pushWait(ctx, sem, n)

	errCountFinal := atomic.LoadUint64(&errCount)
	if errCountFinal > 0 {
		return fmt.Errorf("failed to push %v files", errCountFinal)
	}

	return nil
}

func pushFile(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain string, fileInfo *util.FileInfo, logFn logPushFile) error {
	result := pushError
	defer logFn(fileInfo, &result, time.Now())

	data, err := ioutil.ReadFile(fileInfo.File)
	if err != nil {
		return err
	}

	res, err := util.CreateOrUpdateFile(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, fileInfo.Path, data)
	if err != nil {
		return err
	}

	resStr := util.JSONValue(res, "result").(string)
	switch {
	case strings.Contains(resStr, "File was updated"):
		result = pushOK
	case strings.Contains(resStr, "File was created"):
		result = pushNew
	default:
		result = pushSuccess
	}

	return nil
}

func pushObjects(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain string, reObjects, reIgnoreObjects *regexp.Regexp, pkgs util.PackageSlice, sem *semaphore.Weighted, n int64) error {
	objects, err := util.GetProjectObjects(pkgs)
	if err != nil {
		return err
	}

	matchingObjects := objects[:0]
	maxQNameLength := 0
	for _, objInfo := range objects {
		if !reObjects.MatchString(objInfo.QName) || reIgnoreObjects.MatchString(objInfo.QName) {
			log.DbgLogger2.Println("object ignored:", objInfo.QName)
			continue
		}

		matchingObjects = append(matchingObjects, objInfo)
		if maxQNameLength < len(objInfo.QName) {
			maxQNameLength = len(objInfo.QName)
		}
	}

	maxPkgLength := 0
	for _, pkg := range pkgs {
		if maxPkgLength < len(pkg.Name) {
			maxPkgLength = len(pkg.Name)
		}
	}

	logFn := func(objInfo *util.ObjectInfo, result *pushResult, start time.Time) {
		elapsed := time.Since(start)
		lf := fmt.Sprintf("OBJECT: %%-%ds [%%s] %%%ds [%%s]", maxQNameLength, maxPkgLength+maxPushResultLength-len(objInfo.Package.Name))
		log.OutLogger.Printf(lf, objInfo.QName, objInfo.Package.Name, result.String(), elapsed.Truncate(time.Millisecond).String())
	}

	ctx := context.TODO()
	var errCount uint64

	for _, objInfo := range matchingObjects {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		go func(objInfo *util.ObjectInfo) {
			defer sem.Release(1)

			if err := pushObject(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, objInfo, logFn); err != nil {
				log.ErrLogger.Println("Error:", err.Error())
				atomic.AddUint64(&errCount, 1)
			}
		}(objInfo)
	}

	pushWait(ctx, sem, n)

	errCountFinal := atomic.LoadUint64(&errCount)
	if errCountFinal > 0 {
		return fmt.Errorf("failed to push %v objects", errCountFinal)
	}

	return nil
}

func pushObject(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain string, objInfo *util.ObjectInfo, logFn logPushObject) error {
	result := pushError
	defer logFn(objInfo, &result, time.Now())

	obj, err := util.ReadDataFromFile(objInfo.File)
	if err != nil {
		return err
	}

	err = validateObjectName(objInfo.Name, obj)
	if err != nil {
		return err
	}

	res, err := util.CreateOrUpdateObject(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, objInfo.Class, obj)
	if err != nil {
		errors := util.JSONValue(res, "error")
		if errors != nil {
			return fmt.Errorf("%s\n       %v", err.Error(), errors)
		}

		return err
	}

	resVal := util.JSONValue(res, objInfo.Name)
	if resVal == nil {
		resVal = util.JSONValue(res, strings.Replace(objInfo.Name, " ", "_", -1))
	}

	if resVal == nil {
		return fmt.Errorf("unknown push result")
	}

	switch {
	case strings.Contains(resVal.(string), "Configuration was updated"):
		result = pushOK
	case strings.Contains(resVal.(string), "Configuration was created"):
		result = pushNew
	default:
		result = pushSuccess
	}

	return nil
}

func validateObjectName(name string, obj interface{}) error {
	n := util.JSONValue(obj, "name")
	if n == nil || n.(string) == "" {
		return fmt.Errorf("missing 'name' attribute for object: %s", name)
	}

	if name != n {
		return fmt.Errorf("mismatch: object name: %s, file name: %s", n, name)
	}

	return nil
}

func pushWait(ctx context.Context, sem *semaphore.Weighted, n int64) {
	if err := sem.Acquire(ctx, n); err != nil {
		log.ErrLogger.Println("Error:", err.Error())
		return
	}

	defer sem.Release(n)
}
