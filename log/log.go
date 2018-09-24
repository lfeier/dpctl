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

package log

import (
	"io/ioutil"
	"log"
	"os"
)

var ErrLogger *log.Logger
var OutLogger *log.Logger
var DbgLogger1 *log.Logger
var DbgLogger2 *log.Logger
var DbgLogger3 *log.Logger
var DbgLogger4 *log.Logger
var DbgLogger5 *log.Logger

// DebugLevel stores the current verbosity
var DebugLevel int

func init() {
	ErrLogger = log.New(os.Stderr, "", 0)
	OutLogger = log.New(os.Stdout, "", 0)
	DbgLogger1 = log.New(ioutil.Discard, "[dbg1] ", 0)
	DbgLogger2 = log.New(ioutil.Discard, "[dbg2] ", 0)
	DbgLogger3 = log.New(ioutil.Discard, "[dbg3] ", 0)
	DbgLogger4 = log.New(ioutil.Discard, "[dbg4] ", 0)
	DbgLogger5 = log.New(ioutil.Discard, "[dbg5] ", 0)
}

// SetVebosity enables the debug loggers
func SetVebosity(level int) {
	DebugLevel = level

	if level >= 1 {
		DbgLogger1.SetOutput(os.Stderr)
	}

	if level >= 2 {
		DbgLogger2.SetOutput(os.Stderr)
	}

	if level >= 3 {
		DbgLogger3.SetOutput(os.Stderr)
	}

	if level >= 4 {
		DbgLogger4.SetOutput(os.Stderr)
	}

	if level >= 5 {
		DbgLogger5.SetOutput(os.Stderr)
	}
}
