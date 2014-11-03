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

package admin

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/Unknwon/com"

	"github.com/gpmgo/switch/models"
	"github.com/gpmgo/switch/modules/archive"
	"github.com/gpmgo/switch/modules/middleware"
	"github.com/gpmgo/switch/modules/qiniu"
	"github.com/gpmgo/switch/modules/setting"
)

type ApiRevesion struct {
	Id       int64       `json:"id"`
	Package  *ApiPackage `json:"package"`
	Revision string      `json:"revision"`
	Size     int64       `json:"size"`
	Updated  time.Time   `json:"updated"`
}

type ApiPackage struct {
	Id         int64     `json:"id"`
	ImportPath string    `json:"import_path"`
	Created    time.Time `json:"created"`
}

func ListLargeRevisions(ctx *middleware.Context) {
	revs, err := models.GetLocalRevisions()
	if err != nil {
		ctx.JSON(500, map[string]string{
			"error": fmt.Sprintf("fail to get local revisions: %v", err),
		})
		return
	}

	largeRevs := make([]*ApiRevesion, 0, len(revs)/2)
	for _, rev := range revs {
		pkg, err := models.GetPakcageById(rev.PkgId)
		if err != nil {
			ctx.JSON(500, map[string]string{
				"error": fmt.Sprintf("fail to get package by ID(%d): %v", rev.PkgId, err),
			})
			return
		}

		ext := archive.GetExtension(pkg.ImportPath)
		localPath := path.Join(pkg.ImportPath, rev.Revision)
		fpath := path.Join(setting.ArchivePath, localPath+ext)

		if !com.IsFile(fpath) {
			continue
		}

		// Check archive size.
		f, err := os.Open(fpath)
		if err != nil {
			ctx.JSON(500, map[string]string{
				"error": fmt.Sprintf("fail to open file(%s): %v", fpath, err),
			})
			return
		}
		fi, err := f.Stat()
		if err != nil {
			ctx.JSON(500, map[string]string{
				"error": fmt.Sprintf("fail to get file info(%s): %v", fpath, err),
			})
			return
		}
		// Greater then MAX_UPLOAD_SIZE.
		if fi.Size() > setting.MaxUploadSize<<20 {
			largeRevs = append(largeRevs, &ApiRevesion{
				Id: rev.Id,
				Package: &ApiPackage{
					Id:         pkg.Id,
					ImportPath: pkg.ImportPath,
					Created:    pkg.Created,
				},
				Revision: rev.Revision,
				Size:     fi.Size(),
				Updated:  rev.Updated,
			})
			continue
		}
	}

	ctx.JSON(200, map[string]interface{}{
		"revisions": &largeRevs,
	})
}

func BlockPackage(ctx *middleware.Context) {
	id := ctx.QueryInt64("id")
	pkg, err := models.GetPakcageById(id)
	if err != nil {
		if err == models.ErrPackageNotExist {
			ctx.JSON(404, map[string]string{
				"error": err.Error(),
			})
		} else {
			ctx.JSON(500, map[string]string{
				"error": fmt.Sprintf("fail to get package by ID(%d): %v", id, err),
			})
		}
		return
	}

	revs, err := pkg.GetRevisions()
	if err != nil {
		ctx.JSON(500, map[string]string{
			"error": fmt.Sprintf("fail to get package revisions by ID(%d): %v", id, err),
		})
		return
	}

	// Delete package archives.
	ext := archive.GetExtension(pkg.ImportPath)
	for _, rev := range revs {
		switch rev.Storage {
		case models.QINIU:
			key := pkg.ImportPath + "-" + rev.Revision + ext
			if err = qiniu.DeleteArchive(key); err != nil {
				ctx.JSON(500, map[string]string{
					"error": fmt.Sprintf("fail to delete archive(%s): %v", key, err),
				})
				return
			}
		}
	}
	os.RemoveAll(path.Join(setting.ArchivePath, pkg.ImportPath))

	if err = models.BlockPackage(pkg, revs, ctx.Query("note")); err != nil {
		ctx.JSON(500, map[string]string{
			"error": fmt.Sprintf("fail to block package by ID(%d): %v", id, err),
		})
		return
	}

	ctx.JSON(200, map[string]interface{}{
		"ok": true,
	})
}
