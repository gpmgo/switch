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
	"path"

	"github.com/go-martini/martini"

	"github.com/gpmgo/gopm-registry/models"
	"github.com/gpmgo/gopm-registry/modules/auth"
	"github.com/gpmgo/gopm-registry/modules/log"
	"github.com/gpmgo/gopm-registry/modules/middleware"
	"github.com/gpmgo/gopm-registry/modules/setting"
)

func NewPackage(ctx *middleware.Context) {
	ctx.Data["Title"] = ctx.Locale.Tr("package.new_package")
	ctx.Data["PageIsNewPackage"] = true
	ctx.Data["Categories"] = setting.Categories
	ctx.HTML(200, "package/new")
}

func NewPackagePost(ctx *middleware.Context, form auth.CreatePackageForm) {
	ctx.Data["Title"] = ctx.Locale.Tr("package.new_package")
	ctx.Data["PageIsNewPackage"] = true
	ctx.Data["Categories"] = setting.Categories

	if ctx.HasError() {
		ctx.HTML(200, "package/new")
		return
	}

	if len(form.FullName) == 0 {
		form.FullName = form.PkgName
	}
	pkg := &models.Package{
		OwnerId:     ctx.User.Id,
		Name:        form.PkgName,
		FullName:    form.FullName,
		Description: form.Desc,
		Homepage:    form.Homepage,
		Issues:      form.IssueLink,
		Donate:      form.Donate,
	}
	if err := pkg.SetCategory(form.Category); err != nil {
		ctx.Data["Err_Category"] = true
		ctx.RenderWithErr(err.Error(), "package/new", &form)
		return
	}
	if err := models.NewPackage(pkg); err != nil {
		if err == models.ErrPackageAlreadyExist {
			ctx.Data["Err_PkgName"] = true
			ctx.RenderWithErr(err.Error(), "package/new", &form)
		} else {
			ctx.Handle(500, "pkg.NewPackagePost(NewPackage)", err)
		}
		return
	}
	log.Trace("%s Package created: %s/%s", ctx.Req.RequestURI, ctx.User.UserName, pkg.Name)
	ctx.Redirect("/" + ctx.User.UserName + "/" + pkg.Name)
}

func Download(ctx *middleware.Context, params martini.Params) {
	uname := params["username"]
	pkgname := params["pkgname"]
	u, err := models.GetUserByName(uname)
	if err != nil {
		if err == models.ErrUserNotExist {
			ctx.Handle(404, "pkg.Profile(GetUserByName)", err)
		} else {
			ctx.Handle(500, "pkg.Profile(GetUserByName)", err)
		}
		return
	}
	pkg, err := models.GetPackageByName(u.Id, pkgname)
	if err != nil {
		if err == models.ErrPackageNotExist {
			ctx.Handle(404, "pkg.Profile(GetPackageByName)", err)
		} else {
			ctx.Handle(500, "pkg.Profile(GetPackageByName)", err)
		}
		return
	}
	pkg.GetOwner()
	pkg.GetReleases()

	tag := ctx.Query("r")
	if len(tag) == 0 {
		// Get latest release.
		if len(pkg.Releases) == 0 {
			ctx.Error(404)
			return
		}
		tag = pkg.Releases[0].Tag
	} else {
		isFound := false
		for _, r := range pkg.Releases {
			if r.Tag == tag {
				isFound = true
				break
			}
		}
		if !isFound {
			ctx.Error(404)
			return
		}
	}
	ctx.ServeFile(path.Join(setting.ArchivePath, pkg.Owner.UserName, pkg.Name+"."+tag+".zip"))
}

func Profile(ctx *middleware.Context, params martini.Params) {
	uname := params["username"]
	pkgname := params["pkgname"]
	u, err := models.GetUserByName(uname)
	if err != nil {
		if err == models.ErrUserNotExist {
			ctx.Handle(404, "pkg.Profile(GetUserByName)", err)
		} else {
			ctx.Handle(500, "pkg.Profile(GetUserByName)", err)
		}
		return
	}
	pkg, err := models.GetPackageByName(u.Id, pkgname)
	if err != nil {
		if err == models.ErrPackageNotExist {
			ctx.Handle(404, "pkg.Profile(GetPackageByName)", err)
		} else {
			ctx.Handle(500, "pkg.Profile(GetPackageByName)", err)
		}
		return
	}
	pkg.GetOwner()
	pkg.GetReleases()
	for _, r := range pkg.Releases {
		r.GetGoVersion()
	}

	ctx.Data["Title"] = pkg.FullName
	ctx.Data["Package"] = pkg
	ctx.HTML(200, "package/profile")
}

func EditProfile(ctx *middleware.Context, params martini.Params) {
	ctx.Data["Title"] = ctx.Locale.Tr("edit_profile")
	uname := params["username"]
	pkgname := params["pkgname"]
	if uname != ctx.User.UserName {
		ctx.Error(403)
		return
	}

	pkg, err := models.GetPackageByName(ctx.User.Id, pkgname)
	if err != nil {
		if err == models.ErrPackageNotExist {
			ctx.Handle(404, "pkg.EditProfile(GetPackageByName)", err)
		} else {
			ctx.Handle(500, "pkg.EditProfile(GetPackageByName)", err)
		}
		return
	}
	pkg.GetCategory()
	pkg.Owner = ctx.User
	ctx.Data["Package"] = pkg
	ctx.Data["Categories"] = setting.Categories
	ctx.HTML(200, "package/profile_edit")
}

func EditProfilePost(ctx *middleware.Context, params martini.Params, form auth.CreatePackageForm) {
	ctx.Data["Title"] = ctx.Locale.Tr("edit_profile")
	uname := params["username"]
	pkgname := params["pkgname"]
	if uname != ctx.User.UserName {
		ctx.Error(403)
		return
	}
	pkg, err := models.GetPackageByName(ctx.User.Id, pkgname)
	if err != nil {
		if err == models.ErrPackageNotExist {
			ctx.Handle(404, "pkg.EditProfilePost(GetPackageByName)", err)
		} else {
			ctx.Handle(500, "pkg.EditProfilePost(GetPackageByName)", err)
		}
		return
	}

	pkg.Name = form.PkgName
	pkg.FullName = form.FullName
	pkg.Description = form.Desc
	pkg.Homepage = form.Homepage
	pkg.Issues = form.IssueLink
	pkg.Donate = form.Donate
	pkg.SetCategory(form.Category)
	if err = models.UpdatePackage(pkg); err != nil {
		ctx.Handle(500, "pkg.EditProfilePost(UpdatePackage)", err)
		return
	}

	ctx.Flash.Success("Package profile has been successfully updated.")
	ctx.Redirect("/" + uname + "/" + pkg.Name + "/edit")
}
