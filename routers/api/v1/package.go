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

package v1

import (
	"path"

	"github.com/gpmgo/switch/models"
	"github.com/gpmgo/switch/modules/archive"
	"github.com/gpmgo/switch/modules/base"
	"github.com/gpmgo/switch/modules/middleware"
	"github.com/gpmgo/switch/modules/setting"
)

func Download(ctx *middleware.Context) {
	importPath := archive.GetRootPath(ctx.Query("pkgname"))
	if len(importPath) == 0 {
		ctx.JSON(404, map[string]interface{}{
			"error": "resource not found",
		})
		return
	}

	rev := ctx.Query("revision")
	r, err := models.CheckPkg(importPath, rev)
	if err != nil {
		ctx.JSON(422, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	if err = models.IncreasePackageDownloadCount(importPath); err != nil {
		ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
		return
	} else if err = models.IncreaseRevisionDownloadCount(r.Id); err != nil {
		ctx.JSON(500, map[string]interface{}{
			"error": err.Error(),
		})
		return
	} else {
		if err = models.AddDownloader(ctx.RemoteAddr()); err != nil {
			ctx.JSON(500, map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
	}

	ext := archive.GetExtension(importPath)
	serveName := path.Base(importPath) + "-" + base.ShortSha(r.Revision) + ext
	switch r.Storage {
	case models.LOCAL:
		ctx.ServeFile(path.Join(setting.ArchivePath, importPath, r.Revision+ext), serveName)
	case models.QINIU:
		ctx.Redirect("http://" + setting.BucketUrl + "/" + importPath + "-" + r.Revision)
	}
}
