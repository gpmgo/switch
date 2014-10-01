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
	"fmt"

	"github.com/gpmgo/switch/models"
	"github.com/gpmgo/switch/modules/middleware"
)

func Package(ctx *middleware.Context) {
	importPath := ctx.Params("*")
	_, err := models.GetPakcageByPath(importPath)
	if err != nil {
		if err == models.ErrPackageNotExist {
			ctx.Handle(404, "Package", nil)
		} else {
			ctx.Handle(500, "Package", err)
		}
		return
	}

	ctx.Data["Title"] = importPath
	ctx.Data["ImportPath"] = importPath
	ctx.HTML(200, "package")
}

func Badge(ctx *middleware.Context) {
	importPath := ctx.Params("*")
	pkg, err := models.GetPakcageByPath(importPath)
	if err != nil {
		if err == models.ErrPackageNotExist {
			ctx.Error(404)
		} else {
			ctx.Handle(500, "Badge", err)
		}
		return
	}
	ctx.Redirect(fmt.Sprintf("http://img.shields.io/badge/downloads-%d_total-blue.svg?style=flat", pkg.DownloadCount))
}
