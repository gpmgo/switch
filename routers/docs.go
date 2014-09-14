// Copyright 2014 Unknwon
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package routers

import (
	// "fmt"

	"github.com/gpmgo/switch/modules/base"
	"github.com/gpmgo/switch/modules/middleware"
)

var docUrls = map[string]bool{}

func Docs(ctx *middleware.Context) {
	url := ctx.Params("*")
	if !docUrls[url] {
		ctx.Handle(404, "Documentation", nil)
		return
	}

	ctx.HTML(200, "docs/"+base.TplName(url))
}
