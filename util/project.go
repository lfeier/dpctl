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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/lfeier/dpctl/log"
)

// Package metadata and the source directory
type Package struct {
	Name     string
	Dir      string
	Tags     []string `json:"tags"`
	Priority uint     `json:"priority"`
}

// PackageSlice attaches the methods of the sort Interface to []Package, sorting in decreasing priority order
type PackageSlice []*Package

// Len returns the number of elements in the collection
func (p PackageSlice) Len() int {
	return len(p)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (p PackageSlice) Less(i, j int) bool {
	return p[i].Priority <= p[j].Priority
}

// Swap swaps the elements with indexes i and j.
func (p PackageSlice) Swap(i, j int) {
	t := p[i]
	p[i] = p[j]
	p[j] = t
}

// Sort is a convenience method.
func (p PackageSlice) Sort() {
	sort.Sort(p)
}

// HasTag checks if the package has a given tag
func (p *Package) HasTag(tag string) bool {
	if len(p.Tags) == 0 {
		return false
	}

	for _, t := range p.Tags {
		if tag == t {
			return true
		}
	}

	return false
}

// ProjectPackages retuns all project packages sorted by priority
func ProjectPackages(projectDir string) (PackageSlice, error) {
	var pkgs PackageSlice

	p, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, err
	}

	fs, err := os.Stat(p)
	if err != nil && os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", p)
	}

	if !fs.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", p)
	}

	a, err := filepath.Glob(filepath.Join(p, "*", "metadata.json"))
	if err != nil {
		return nil, err
	}

	for _, f := range a {
		fs, err := os.Stat(f)
		if err != nil {
			return nil, err
		}

		if fs.IsDir() {
			continue
		}

		j, err := ioutil.ReadFile(f)
		if err != nil {
			return nil, err
		}

		pkg := &Package{
			Name: filepath.Base(filepath.Dir(f)),
			Dir:  filepath.Dir(f),
		}

		if err := json.Unmarshal(j, &pkg); err != nil {
			return nil, err
		}

		pkgs = append(pkgs, pkg)
	}

	pkgs.Sort()

	return pkgs, nil
}

// FilterPackages returns all packages matching the tags sorted by priority
func FilterPackages(pkgs PackageSlice, tags []string) PackageSlice {
	var s PackageSlice

	for _, pkg := range pkgs {
		for _, t := range tags {
			if !pkg.HasTag(t) {
				goto Next
			}
		}

		s = append(s, pkg)
	Next:
	}

	s.Sort()

	return s
}

// ObjectQName returns a qualified name: objclass/objname
func ObjectQName(cls string, name string) string {
	return fmt.Sprintf("%s/%s", cls, name)
}

// GetObjectPackage returns the package where the object is saved
func GetObjectPackage(pkgs PackageSlice, qname string) (*Package, error) {
	for _, pkg := range pkgs {
		f := filepath.Join(pkg.Dir, "objects", fmt.Sprintf("%s.json", qname))

		fs, err := os.Stat(f)
		if os.IsNotExist(err) {
			continue
		}

		if err != nil {
			return nil, err
		}

		if fs.IsDir() {
			return nil, fmt.Errorf("invalid package, %s is a directory", f)
		}

		return pkg, nil
	}

	return nil, nil
}

// GetFilePackage returns the package where the file is saved
func GetFilePackage(pkgs PackageSlice, path string) (*Package, error) {
	for _, pkg := range pkgs {
		f := filepath.Join(pkg.Dir, "files", path)

		fs, err := os.Stat(f)
		if os.IsNotExist(err) {
			continue
		}

		if err != nil {
			return nil, err
		}

		if fs.IsDir() {
			return nil, fmt.Errorf("invalid package, %s is a directory", f)
		}

		return pkg, nil
	}

	return nil, nil
}

// SaveObject writes the configuration object to a file
func SaveObject(pkgDir string, qname string, obj interface{}) (string, bool, error) {
	new := false

	p, err := filepath.Abs(pkgDir)
	if err != nil {
		return "", new, err
	}

	f := filepath.Join(p, "objects", fmt.Sprintf("%s.json", qname))
	if err := os.MkdirAll(filepath.Dir(f), 0777); err != nil {
		return "", new, err
	}

	_, err = os.Stat(f)
	if err != nil {
		if os.IsNotExist(err) {
			new = true
		} else {
			return "", new, err
		}
	}

	if err := WriteDataToFile(obj, f); err != nil {
		return "", new, err
	}

	return f, new, nil
}

// SaveFile writes the configuration file
func SaveFile(pkgDir string, path string, data []byte) (string, bool, error) {
	new := false

	p, err := filepath.Abs(pkgDir)
	if err != nil {
		return "", new, err
	}

	f := filepath.Join(p, "files", path)
	if err := os.MkdirAll(filepath.Dir(f), 0777); err != nil {
		return "", new, err
	}

	_, err = os.Stat(f)
	if err != nil {
		if os.IsNotExist(err) {
			new = true
		} else {
			return "", new, err
		}
	}

	if err := ioutil.WriteFile(f, data, 0644); err != nil {
		return "", new, err
	}

	return f, new, nil
}

// ObjectInfo describes a project object
type ObjectInfo struct {
	Name    string
	Class   string
	QName   string
	Package *Package
	File    string
}

// GetProjectObjects returns the project objects for the selected packages
func GetProjectObjects(pkgs PackageSlice) ([]*ObjectInfo, error) {
	pkgs.Sort()

	m := make(map[string]*ObjectInfo)

	var objectsDir string
	var cls string
	var pkg *Package
	var objInfo *ObjectInfo

	walkFn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if IsHidden(path) {
				return filepath.SkipDir
			}

			if path == objectsDir {
				return nil
			}

			if filepath.Dir(path) != objectsDir {
				return fmt.Errorf("unexpected package directory: %v", path)
			}

			cls = info.Name()

			return nil
		}

		if IsHidden(path) {
			return nil
		}

		if filepath.Dir(path) == objectsDir {
			return fmt.Errorf("unexpected package file: %v", path)
		}

		n := filepath.Base(path)

		if filepath.Ext(n) != ".json" {
			return fmt.Errorf("object file must have the 'json' extension: %v", path)
		}

		objInfo = &ObjectInfo{
			Name:    n[0 : len(n)-5],
			Class:   cls,
			Package: pkg,
			File:    path,
		}

		objInfo.QName = ObjectQName(objInfo.Class, objInfo.Name)

		if _, ok := m[objInfo.QName]; !ok {
			m[objInfo.QName] = objInfo
		} else {
			log.DbgLogger4.Printf("package object ignored: %s [%s]", objInfo.QName, objInfo.Package.Name)
		}

		return nil
	}

	for _, pkg = range pkgs {
		objectsDir = filepath.Join(pkg.Dir, "objects")

		fs, err := os.Stat(objectsDir)
		if err != nil && os.IsNotExist(err) {
			continue
		}
		if !fs.IsDir() {
			return nil, fmt.Errorf("not a directory: %s", objectsDir)
		}

		if err := filepath.Walk(objectsDir, walkFn); err != nil {
			return nil, err
		}
	}

	objects := make([]*ObjectInfo, 0, len(m))

	for _, objInfo := range m {
		objects = append(objects, objInfo)
	}

	return objects, nil
}

// FileInfo describes a project object
type FileInfo struct {
	Path    string
	Package *Package
	File    string
}

// GetProjectFiles returns the project files for the selected packages
func GetProjectFiles(pkgs PackageSlice) ([]*FileInfo, error) {
	pkgs.Sort()

	m := make(map[string]*FileInfo)

	var filesDir string
	var pkg *Package
	var fileInfo *FileInfo

	walkFn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if IsHidden(path) {
				return filepath.SkipDir
			}

			if path == filesDir {
				return nil
			}

			return nil
		}

		if IsHidden(path) {
			return nil
		}

		if filepath.Dir(path) == filesDir {
			return fmt.Errorf("unexpected package file: %v", path)
		}

		rel, err := filepath.Rel(filesDir, path)
		if err != nil {
			return err
		}

		fileInfo = &FileInfo{
			Path:    rel,
			Package: pkg,
			File:    path,
		}

		if _, ok := m[rel]; !ok {
			m[rel] = fileInfo
		} else {
			log.DbgLogger4.Printf("package file ignored: %s [%s]", path, fileInfo.Package.Name)
		}

		return nil
	}

	for _, pkg = range pkgs {
		filesDir = filepath.Join(pkg.Dir, "files")

		fs, err := os.Stat(filesDir)
		if err != nil && os.IsNotExist(err) {
			continue
		}
		if !fs.IsDir() {
			return nil, fmt.Errorf("not a directory: %s", filesDir)
		}

		if err := filepath.Walk(filesDir, walkFn); err != nil {
			return nil, err
		}
	}

	files := make([]*FileInfo, 0, len(m))

	for _, fileInfo := range m {
		files = append(files, fileInfo)
	}

	return files, nil
}

// IsHidden return true for a hidden file or directory, false otherwise
func IsHidden(path string) bool {
	if filepath.Base(path)[0:1] == "." {
		return true
	}

	// TODO: Update to handle hidden Windows files and directories

	return false
}
