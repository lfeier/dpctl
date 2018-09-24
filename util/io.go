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
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"
)

// OutputData prints the data to the stdout
func OutputData(data interface{}, format string, templateFile string) error {
	if templateFile != "" {
		return templateOutput(data, templateFile)
	}

	switch format {
	case "json":
		return jsonOutput(data)
	case "yaml":
		return yamlOutput(data)
	case "xml":
		return xmlOutput(data)
	case "txt":
		return textOutput(data)
	default:
		return fmt.Errorf("unknown output format: %s", format)
	}
}

// ReadDataFromFile reads JSON data from a file
func ReadDataFromFile(file string) (interface{}, error) {
	if file == "" {
		return nil, fmt.Errorf("input file not specified")
	}

	j, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var data interface{}
	if err := json.Unmarshal(j, &data); err != nil {
		return nil, err
	}

	return data, nil
}

// WriteDataToFile writes the data to a file in JSON format
func WriteDataToFile(data interface{}, file string) error {
	b, err := json.MarshalIndent(data, "", "  ")

	if err != nil {
		return err
	}

	return ioutil.WriteFile(file, b, 0644)
}

func templateOutput(data interface{}, templateFile string) error {
	t, err := template.ParseFiles(templateFile)
	if err != nil {
		return fmt.Errorf("failed to open template file: %s", err.Error())
	}

	w := bufio.NewWriter(os.Stdout)
	if err = t.Execute(w, data); err != nil {
		return err
	}

	if err = w.Flush(); err != nil {
		return err
	}

	return nil
}

func jsonOutput(data interface{}) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))

	return nil
}

func yamlOutput(data interface{}) error {
	return fmt.Errorf("output format not implemented: yaml")
}

func xmlOutput(data interface{}) error {
	return fmt.Errorf("output format not implemented: xml")
}

func textOutput(data interface{}) error {
	return fmt.Errorf("output format not implemented: txt")
}
