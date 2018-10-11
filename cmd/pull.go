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
	"net/http"
	"reflect"
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
		Use:    "pull",
		Short:  "Pull DataPower configuration objects and files",
		Long:   ``,
		PreRun: preRunPull,
		Run:    runPull,
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

func preRunPull(cmd *cobra.Command, args []string) {
	level, _ := getVerboseFlagValue(cmd)
	log.SetVebosity(level)
}

func runPull(cmd *cobra.Command, args []string) {
	if err := runPullE(cmd, args); err != nil {
		log.ErrLogger.Println("Error:", err.Error())
	}
}

func runPullE(cmd *cobra.Command, args []string) error {
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

	err1 := pullFiles(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, reFiles, reIgnoreFiles, pkgs, sem, int64(parallel))

	err2 := pullObjects(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, reObjects, reIgnoreObjects, pkgs, sem, int64(parallel))

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

type pullResult int

const (
	pullError pullResult = iota
	pullOK
	pullNew
	pullSuccess
	pullDryRun
)

func (result *pullResult) String() string {
	names := [...]string{
		"ERROR",
		"OK",
		"NEW",
		"SUCCESS",
		"DRYRUN",
	}

	return names[*result]
}

var maxPullResultLength = 7

type logPullFile func(fileInfo *util.FileInfo, result *pullResult, start time.Time)
type logPullObject func(objectInfo *util.ObjectInfo, result *pullResult, start time.Time)

func pullFiles(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain string, reFiles, reIgnoreFiles *regexp.Regexp, pkgs util.PackageSlice, sem *semaphore.Weighted, n int64) error {
	walkDir := func(path string) error {
		if reIgnoreFiles.MatchString(path) || reIgnoreFiles.MatchString(fmt.Sprintf("%s/", path)) {
			log.DbgLogger2.Println("directory ignored:", path)
			return util.ErrSkipDir
		}

		return nil
	}

	var files util.FileInfoSlice
	maxPathLength := 0
	maxPkgLength := 0
	walkFile := func(path string, modified string, size uint) error {
		if !reFiles.MatchString(path) || reIgnoreFiles.MatchString(path) {
			log.DbgLogger2.Println("file ignored:", path)
			return nil
		}

		pkg, err := util.GetFilePackage(pkgs, path)
		if err != nil {
			return err
		}

		if pkg == nil {
			pkg = pkgs[0]
		}

		fileInfo := &util.FileInfo{
			Path:    path,
			Package: pkg,
		}

		files = append(files, fileInfo)

		if maxPathLength < len(path) {
			maxPathLength = len(path)
		}

		if maxPkgLength < len(pkg.Name) {
			maxPkgLength = len(pkg.Name)
		}

		return nil
	}

	stores, err := util.GetFileStores(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain)
	if err != nil {
		return err
	}

	for _, store := range stores {
		if store == "cert" || store == "sharedcert" || store == "pubcert" {
			log.DbgLogger2.Println("store ignored:", store)
			continue
		}

		err = util.WalkFileStore(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, store, walkDir, walkFile)
		if err != nil {
			return err
		}
	}

	logFn := func(fileInfo *util.FileInfo, result *pullResult, start time.Time) {
		elapsed := time.Since(start)
		lf := fmt.Sprintf("FILE: %%-%ds [%%s] %%%ds [%%s]", maxPathLength, maxPkgLength+maxPullResultLength-len(fileInfo.Package.Name))
		log.OutLogger.Printf(lf, fileInfo.Path, fileInfo.Package.Name, result.String(), elapsed.Truncate(time.Millisecond).String())
	}

	log.DbgLogger1.Printf("files selected: %d", len(files))

	ctx := context.TODO()
	var errCount uint64

	for _, fileInfo := range files {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		go func(fileInfo *util.FileInfo) {
			defer sem.Release(1)

			if err := pullFile(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, fileInfo, logFn); err != nil {
				log.ErrLogger.Println("Error:", err.Error())
				atomic.AddUint64(&errCount, 1)
			}
		}(fileInfo)
	}

	pullWait(ctx, sem, n)

	errCountFinal := atomic.LoadUint64(&errCount)
	if errCountFinal > 0 {
		return fmt.Errorf("failed to pull %v files", errCountFinal)
	}

	return nil
}

func pullFile(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain string, fileInfo *util.FileInfo, logFn logPullFile) error {
	result := pullError
	defer logFn(fileInfo, &result, time.Now())

	data, err := util.GetFile(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, fileInfo.Path)
	if err != nil {
		return err
	}

	f, new, err := util.SaveFile(fileInfo.Package.Dir, fileInfo.Path, data)
	if err != nil {
		return err
	}

	log.DbgLogger4.Println("file local path:", f)

	if new {
		result = pullNew
	} else {
		result = pullOK
	}

	return nil
}

func pullObjects(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain string, reObjects, reIgnoreObjects *regexp.Regexp, pkgs util.PackageSlice, sem *semaphore.Weighted, n int64) error {
	res, err := util.GetStatus(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, "ObjectStatus")
	if err != nil {
		return err
	}

	var objects util.ObjectInfoSlice
	maxQNameLength := 0
	maxPkgLength := 0
	for _, objStatus := range util.JSONValue(res, "ObjectStatus").([]interface{}) {
		name := util.JSONValue(objStatus, "Name").(string)
		cls := util.JSONValue(objStatus, "Class").(string)
		qn := util.ObjectQName(cls, name)

		if !reObjects.MatchString(qn) || reIgnoreObjects.MatchString(qn) {
			log.DbgLogger2.Println("object ignored:", qn)
			continue
		}

		pkg, err := util.GetObjectPackage(pkgs, qn)
		if err != nil {
			return err
		}

		if pkg == nil {
			pkg = pkgs[0]
		}

		objInfo := &util.ObjectInfo{
			Name:    name,
			Class:   cls,
			Package: pkg,
		}

		objects = append(objects, objInfo)

		if maxQNameLength < len(qn) {
			maxQNameLength = len(qn)
		}

		if maxPkgLength < len(pkg.Name) {
			maxPkgLength = len(pkg.Name)
		}
	}

	logFn := func(objInfo *util.ObjectInfo, result *pullResult, start time.Time) {
		elapsed := time.Since(start)
		lf := fmt.Sprintf("OBJECT: %%-%ds [%%s] %%%ds [%%s]", maxQNameLength, maxPkgLength+maxPullResultLength-len(objInfo.Package.Name))
		log.OutLogger.Printf(lf, objInfo.QName(), objInfo.Package.Name, result.String(), elapsed.Truncate(time.Millisecond).String())
	}

	log.DbgLogger1.Printf("objects selected: %d", len(objects))

	ctx := context.TODO()
	var errCount uint64

	for _, objInfo := range objects {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		go func(objInfo *util.ObjectInfo) {
			defer sem.Release(1)

			if err := pullObject(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, objInfo, logFn); err != nil {
				log.ErrLogger.Println("Error:", err.Error())
				atomic.AddUint64(&errCount, 1)
			}
		}(objInfo)
	}

	pullWait(ctx, sem, n)

	errCountFinal := atomic.LoadUint64(&errCount)
	if errCountFinal > 0 {
		return fmt.Errorf("failed to pull %v objects", errCountFinal)
	}

	return nil
}

func pullObject(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain string, objInfo *util.ObjectInfo, logFn logPullObject) error {
	result := pullError
	defer logFn(objInfo, &result, time.Now())

	obj, err := util.GetObject(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, objInfo.Class, objInfo.Name)
	if err != nil && strings.Contains(err.Error(), "HTTP response error: 404 Not Found") {
		obj, err = util.GetSingletonObject(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, objInfo.Class)
	}
	if err != nil {
		return err
	}

	name := util.JSONValue(obj, "name").(string)
	if objInfo.Name != name {
		objInfo.Name = name
	}

	updateLinks(obj.(util.GenericMap), domain)

	f, new, err := util.SaveObject(objInfo.Package.Dir, objInfo.QName(), obj)
	if err != nil {
		return err
	}

	log.DbgLogger4.Println("object local path:", f)

	if new {
		result = pullNew
	} else {
		result = pullOK
	}

	return nil
}

func updateLinks(o util.GenericMap, domain string) {
	for k, v := range o {
		switch k {
		case "_links":
			delete(o, k)
			continue
		case "href":
			o[k] = strings.Replace(v.(string), fmt.Sprintf("/mgmt/config/%s/", domain), "/mgmt/config/{domain}/", 1)
			continue
		}

		switch reflect.ValueOf(v).Kind() {
		case reflect.Map:
			updateLinks(v.(util.GenericMap), domain)
		case reflect.Slice:
			for _, sv := range v.([]interface{}) {
				if reflect.ValueOf(sv).Kind() == reflect.Map {
					updateLinks(sv.(util.GenericMap), domain)
				}
			}
		}
	}
}

func pullWait(ctx context.Context, sem *semaphore.Weighted, n int64) {
	if err := sem.Acquire(ctx, n); err != nil {
		log.ErrLogger.Println("Error:", err.Error())
		return
	}

	defer sem.Release(n)
}
