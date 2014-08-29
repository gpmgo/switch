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

package pkg

import (
	"io"
	"os"
	"path"
	"strings"

	"github.com/go-martini/martini"

	"github.com/gpmgo/gopm-registry/models"
	"github.com/gpmgo/gopm-registry/modules/middleware"
	"github.com/gpmgo/gopm-registry/modules/setting"
)

func NewRelease(ctx *middleware.Context, params martini.Params) {
	ctx.Data["Title"] = ctx.Locale.Tr("package.new_release")
	uname := params["username"]
	pkgname := params["pkgname"]
	if uname != ctx.User.UserName {
		ctx.Error(403)
		return
	}

	pkg, err := models.GetPackageByName(ctx.User.Id, pkgname)
	if err != nil {
		if err == models.ErrPackageNotExist {
			ctx.Handle(404, "release.NewRelease(GetPackageByName)", err)
		} else {
			ctx.Handle(500, "release.NewRelease(GetPackageByName)", err)
		}
		return
	}
	pkg.Owner = ctx.User
	ctx.Data["Package"] = pkg
	ctx.Data["GoVersions"] = setting.GoVersions
	ctx.HTML(200, "package/release_new")
}

func NewReleasePost(ctx *middleware.Context, params martini.Params) {
	ctx.Data["Title"] = ctx.Locale.Tr("package.new_release")
	uname := params["username"]
	pkgname := params["pkgname"]
	if uname != ctx.User.UserName {
		ctx.Error(403)
		return
	}

	pkg, err := models.GetPackageByName(ctx.User.Id, pkgname)
	if err != nil {
		if err == models.ErrPackageNotExist {
			ctx.Handle(404, "release.NewReleasePost(GetPackageByName)", err)
		} else {
			ctx.Handle(500, "release.NewReleasePost(GetPackageByName)", err)
		}
		return
	}
	pkg.Owner = ctx.User
	ctx.Data["Package"] = pkg
	ctx.Data["GoVersions"] = setting.GoVersions
	if ctx.HasError() {
		ctx.HTML(200, "package/release_new")
		return
	}

	// Parse form.
	ctx.Req.ParseMultipartForm(1 << 22) // 4MB

	tag := ctx.Query("release_name")
	if len(tag) == 0 {
		ctx.Data["Err_ReleaseName"] = true
		ctx.RenderWithErr("Release tag name cannot be empty", "package/release_new", nil)
		return
	} else if len(tag) > 35 {
		ctx.Data["Err_ReleaseName"] = true
		ctx.RenderWithErr("Release tag name must contain at most 35 characters.", "package/release_new", nil)
		return
	}

	r := &models.Release{
		PkgId:  pkg.Id,
		Tag:    strings.Replace(tag, " ", "", -1),
		Source: models.REGISTRY,
	}
	if err = r.SetGoVersion(ctx.Query("gover")); err != nil {
		ctx.Data["Err_GoVer"] = true
		ctx.RenderWithErr(err.Error(), "package/release_new", nil)
		return
	} else if err = models.NewRelease(pkg, r); err != nil {
		ctx.Handle(500, "release.NewReleasePost(NewRelease)", err)
		return
	}

	file, _, err := ctx.Req.FormFile("source")
	if err != nil {
		ctx.Handle(500, "release.NewReleasePost(FormFile)", err)
		return
	}
	defer file.Close()

	archivePath := path.Join(setting.ArchivePath, uname, pkgname+"."+r.Tag+".zip")
	os.MkdirAll(path.Dir(archivePath), os.ModePerm)
	f, err := os.Create(archivePath)
	if err != nil {
		ctx.Handle(500, "release.NewReleasePost(os.Create)", err)
		return
	}
	defer f.Close()
	if _, err = io.Copy(f, file); err != nil {
		ctx.Handle(500, "release.NewReleasePost(io.Copy)", err)
		return
	}
	ctx.Redirect("/" + uname + "/" + pkgname)
}
