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
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/lfeier/dpctl/log"
)

// CreateHTTPClient creates an HTTP client
func CreateHTTPClient(httpTimeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport)
	transport.DialContext = (&net.Dialer{
		Timeout:   httpTimeout,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return &http.Client{
		Transport: transport,
	}
}

// AbsoluteMgmtURL returns the absolute REST management URL
// after substituting path placeholders
func AbsoluteMgmtURL(rootMgmtURL, mgmtURL string, a ...interface{}) (string, error) {
	u, err := url.Parse(rootMgmtURL)
	if err != nil {
		return "", err
	}

	p := fmt.Sprintf(mgmtURL, a...)

	u, err = u.Parse(p)
	if err != nil {
		return "", err
	}

	return u.String(), nil
}

// DoHTTPRequest sends an HTTP request to DataPower returning the parsed JSON response
func DoHTTPRequest(httpClient *http.Client, method, url, userName, userPassword string, rqBody interface{}) (interface{}, error) {
	var r io.Reader
	if rqBody != nil {
		b, err := json.Marshal(rqBody)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, r)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(userName, userPassword)

	if log.DebugLevel >= 5 {
		dump, _ := httputil.DumpRequestOut(req, true)
		log.DbgLogger5.Printf("HTTP Request:\n%v", string(dump))
	}

	res, err := httpClient.Do(req)

	if log.DebugLevel >= 5 {
		dump, _ := httputil.DumpResponse(res, true)
		log.DbgLogger5.Printf("HTTP Response:\n%v", string(dump))
	}

	if err != nil {
		return nil, err
	}

	defer func() {
		if err := res.Body.Close(); err != nil {
			panic(err.Error())
		}
	}()

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	var rsBody interface{}
	if err := json.Unmarshal(body, &rsBody); err != nil {
		return nil, err
	}

	if res.StatusCode >= 300 {
		return rsBody, fmt.Errorf("HTTP response error: %s", res.Status)
	}

	return rsBody, nil
}

// GetObjectClasses returns all object classes
func GetObjectClasses(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword string) ([]string, error) {
	u, err := AbsoluteMgmtURL(dpRestMgmtURL, "/mgmt/config/")
	if err != nil {
		return nil, err
	}

	rsBody, err := DoHTTPRequest(httpClient, "GET", u, dpUserName, dpUserPassword, nil)
	if err != nil {
		return nil, err
	}

	l := JSONValue(rsBody, "_links").(map[string]interface{})

	var s []string
	for c, _ := range l {
		if c == "self" {
			continue
		}
		s = append(s, c)
	}

	return s, nil
}

// GetObject returns a domain object of a given class and name
func GetObject(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain, class, name string) (interface{}, error) {
	u, err := AbsoluteMgmtURL(dpRestMgmtURL, "/mgmt/config/%s/%s/%s", domain, class, name)
	if err != nil {
		return nil, err
	}

	rsBody, err := DoHTTPRequest(httpClient, "GET", u, dpUserName, dpUserPassword, nil)
	if err != nil {
		return nil, err
	}

	return JSONValue(rsBody, class), nil
}

// GetObjects returns all domain objects of a given class
func GetObjects(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain, class string) ([]interface{}, error) {
	u, err := AbsoluteMgmtURL(dpRestMgmtURL, "/mgmt/config/%s/%s", domain, class)
	if err != nil {
		return nil, err
	}

	rsBody, err := DoHTTPRequest(httpClient, "GET", u, dpUserName, dpUserPassword, nil)
	if err != nil {
		return nil, err
	}

	var s []interface{}

	l := JSONValue(rsBody, class)
	if l == nil {
		return s, nil
	}

	switch reflect.ValueOf(l).Kind() {
	case reflect.Map:
		s = append(s, l)
	case reflect.Slice:
		s = append(s, l.([]interface{})...)
	}

	return s, nil
}

// GetFileStores returns all file stores
func GetFileStores(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain string) ([]string, error) {
	u, err := AbsoluteMgmtURL(dpRestMgmtURL, "/mgmt/filestore/%s", domain)
	if err != nil {
		return nil, err
	}

	rsBody, err := DoHTTPRequest(httpClient, "GET", u, dpUserName, dpUserPassword, nil)
	if err != nil {
		return nil, err
	}

	l := JSONValue(rsBody, "filestore", "location").([]interface{})

	var s []string
	for _, c := range l {
		s = append(s, strings.TrimRight(JSONValue(c, "name").(string), ":"))
	}

	return s, nil
}

// GetFile retrieves a file from the file store
func GetFile(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain, path string) ([]byte, error) {
	u, err := AbsoluteMgmtURL(dpRestMgmtURL, "/mgmt/filestore/%s/%s", domain, path)
	if err != nil {
		return nil, err
	}

	rsBody, err := DoHTTPRequest(httpClient, "GET", u, dpUserName, dpUserPassword, nil)
	if err != nil {
		return nil, err
	}

	fd := JSONValue(rsBody, "file").(string)
	return base64.StdEncoding.DecodeString(fd)
}

// CreateOrUpdateObject creates a configuration object or updates it if already exist
func CreateOrUpdateObject(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain, cls string, obj interface{}) (interface{}, error) {
	u, err := AbsoluteMgmtURL(dpRestMgmtURL, "/mgmt/config/%s/%s/%s", domain, cls, JSONValue(obj, "name").(string))
	if err != nil {
		return nil, err
	}

	m := make(map[string]interface{})
	m[cls] = obj

	rsBody, err := DoHTTPRequest(httpClient, "PUT", u, dpUserName, dpUserPassword, m)
	if err != nil {
		return nil, err
	}

	return rsBody, nil
}

// IsDirectory checks if a directory exist
func IsDirectory(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain, path string) (bool, error) {
	u, err := AbsoluteMgmtURL(dpRestMgmtURL, "/mgmt/filestore/%s/%s", domain, path)
	if err != nil {
		return false, err
	}

	_, err = DoHTTPRequest(httpClient, "GET", u, dpUserName, dpUserPassword, nil)
	if err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}

// CreateDirectories recursively creates directories
func CreateDirectories(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain, path string) error {
	ok, err := IsDirectory(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, path)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	if err := CreateDirectories(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, filepath.Dir(path)); err != nil {
		return err
	}

	u, err := AbsoluteMgmtURL(dpRestMgmtURL, "/mgmt/filestore/%s/%s", domain, path)
	if err != nil {
		return err
	}

	m := make(map[string]interface{})
	d := make(map[string]interface{})
	m["directory"] = d
	d["name"] = filepath.Base(path)

	_, err = DoHTTPRequest(httpClient, "PUT", u, dpUserName, dpUserPassword, m)
	if err != nil && !strings.Contains(err.Error(), "409 Conflict") {
		return err
	}

	return nil
}

// CreateOrUpdateFile creates or updates a file with the given data
func CreateOrUpdateFile(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain, path string, data []byte) (interface{}, error) {
	if err := CreateDirectories(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, filepath.Dir(path)); err != nil {
		return nil, err
	}

	u, err := AbsoluteMgmtURL(dpRestMgmtURL, "/mgmt/filestore/%s/%s", domain, path)
	if err != nil {
		return nil, err
	}

	m := make(map[string]interface{})
	f := make(map[string]interface{})
	m["file"] = f
	f["name"] = filepath.Base(path)
	f["content"] = base64.StdEncoding.EncodeToString(data)

	rsBody, err := DoHTTPRequest(httpClient, "PUT", u, dpUserName, dpUserPassword, m)
	if err != nil {
		return rsBody, err
	}

	return rsBody, nil
}

// WalkDirFunc is the type of the function called for each directory visited by Walk
type WalkDirFunc func(path string) error

// WalkFileFunc is the type of the function called for each file visited by Walk
type WalkFileFunc func(path string, modified string, size uint) error

// ErrSkipDir skips the directory
var ErrSkipDir = errors.New("skip directory")

// WalkFileStore walks file store rooted at path
func WalkFileStore(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain, path string, walkDirFn WalkDirFunc, walkFileFn WalkFileFunc) error {
	if err := walkDirFn(path); err != nil {
		if err == ErrSkipDir {
			return nil
		} else {
			return err
		}
	}

	var fn func(string) error
	fn = func(p string) error {
		d, f, err := lsFileStore(httpClient, dpRestMgmtURL, dpUserName, dpUserPassword, domain, p)
		if err != nil {
			return err
		}

		for _, a := range f {
			n := JSONValue(a, "name").(string)
			m := JSONValue(a, "modified").(string)
			s := uint(JSONValue(a, "size").(float64))

			if err = walkFileFn(fmt.Sprintf("%s/%s", p, n), m, s); err != nil {
				return err
			}
		}

		for _, a := range d {
			n := JSONValue(a, "name").(string)
			i := strings.LastIndex(n, "/")
			n = n[i+1:]

			if err = walkDirFn(fmt.Sprintf("%s/%s", p, n)); err != nil {
				if err == ErrSkipDir {
					return nil
				} else {
					return err
				}
			}

			if err = fn(fmt.Sprintf("%s/%s", p, n)); err != nil {
				return err
			}
		}

		return nil
	}

	return fn(path)
}

// lsFileStore lists directories and files at a given path
func lsFileStore(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain, path string) (d []interface{}, f []interface{}, e error) {
	u, err := AbsoluteMgmtURL(dpRestMgmtURL, "/mgmt/filestore/%s/%s", domain, path)
	if err != nil {
		return d, f, err
	}

	rsBody, err := DoHTTPRequest(httpClient, "GET", u, dpUserName, dpUserPassword, nil)
	if err != nil {
		return d, f, err
	}

	rf := JSONValue(rsBody, "filestore", "location", "file")

	switch reflect.ValueOf(rf).Kind() {
	case reflect.Map:
		f = append(f, rf)
	case reflect.Slice:
		f = append(f, rf.([]interface{})...)
	}

	rd := JSONValue(rsBody, "filestore", "location", "directory")

	switch reflect.ValueOf(rd).Kind() {
	case reflect.Map:
		d = append(d, rd)
	case reflect.Slice:
		d = append(d, rd.([]interface{})...)
	}

	return d, f, nil
}

// GetStatus returns the status information from a given provider
func GetStatus(httpClient *http.Client, dpRestMgmtURL, dpUserName, dpUserPassword, domain, statusProvider string) (interface{}, error) {
	u, err := AbsoluteMgmtURL(dpRestMgmtURL, "/mgmt/status/%s/%s", domain, statusProvider)
	if err != nil {
		return nil, err
	}

	rsBody, err := DoHTTPRequest(httpClient, "GET", u, dpUserName, dpUserPassword, nil)
	if err != nil {
		return nil, err
	}

	return rsBody, nil
}
