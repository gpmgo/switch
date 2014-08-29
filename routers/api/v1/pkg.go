// Copyright 2014 Unknown
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

package v1

import (
	"github.com/go-martini/martini"

	"github.com/gpmgo/gopm-registry/models"
	"github.com/gpmgo/gopm-registry/modules/middleware"
)

func GetLatestRelease(ctx *middleware.Context, params martini.Params) {
	uname := params["username"]
	u, err := models.GetUserByName(uname)
	if err != nil {
		if err == models.ErrUserNotExist {
			ctx.JSON(404, map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			ctx.JSON(500, map[string]interface{}{
				"error": err.Error(),
			})
		}
		return
	}

	pkgname := params["pkgname"]
	pkg, err := models.GetPackageByName(u.Id, pkgname)
	if err != nil {
		if err == models.ErrPackageNotExist {
			ctx.JSON(404, map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			ctx.JSON(500, map[string]interface{}{
				"error": err.Error(),
			})
		}
		return
	}
	pkg.GetReleases()

	rel := new(models.Release)
	if len(pkg.Releases) > 0 {
		rel = pkg.Releases[0]
	}
	ctx.JSON(200, rel)
}

func GetPkgInfo(ctx *middleware.Context, params martini.Params) {

}
