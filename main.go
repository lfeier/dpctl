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

package main

import (
	"os"
	"time"

	"github.com/lfeier/dpctl/cmd"
	"github.com/lfeier/dpctl/log"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.ErrLogger.Println(r)
			os.Exit(2)
		}
	}()

	defer func(start time.Time) {
		log.DbgLogger1.Printf("Total time: %v", time.Since(start))
	}(time.Now())

	if err := cmd.CmdRoot.Execute(); err != nil {
		log.ErrLogger.Println(err.Error())
		os.Exit(1)
	}
}
