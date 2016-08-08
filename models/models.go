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

package models

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/Unknwon/com"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"github.com/robfig/cron"

	"github.com/gpmgo/switch/modules/archive"
	"github.com/gpmgo/switch/modules/log"
	"github.com/gpmgo/switch/modules/qiniu"
	"github.com/gpmgo/switch/modules/setting"
)

var (
	x *xorm.Engine
)

func init() {
	var err error
	x, err = xorm.NewEngine("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8",
		setting.Cfg.Section("database").Key("USER").String(),
		setting.Cfg.Section("database").Key("PASSWD").String(),
		setting.Cfg.Section("database").Key("HOST").String(),
		setting.Cfg.Section("database").Key("NAME").String()))
	if err != nil {
		log.Fatal(4, "Fail to init new engine: %v", err)
	}

	x.SetMapper(core.GonicMapper{})

	if setting.ProdMode {
		x.SetLogger(xorm.NewSimpleLogger(ioutil.Discard))
	}

	if err = x.Sync2(new(Package), new(Revision), new(Downloader),
		new(Block), new(BlockRule)); err != nil {
		log.Fatal(4, "Fail to sync database: %v", err)
	}

	statistic()
	c := cron.New()
	c.AddFunc("@every 5m", statistic)
	c.AddFunc("@every 1h", cleanExpireRevesions)
	c.Start()

	go cleanExpireRevesions()
	if setting.ProdMode {
		go uploadArchives()
		ticker := time.NewTicker(time.Hour)
		go func() {
			for _ = range ticker.C {
				uploadArchives()
			}
		}()
	}
}

func Ping() error {
	return x.Ping()
}

type DownloadStats struct {
	NumTotalDownload int64
}

type Stats struct {
	NumPackages, NumDownloaders int64
	DownloadStats
	TrendingPackages, NewPackages, PopularPackages []*Package
}

var Statistic Stats

func statistic() {
	var totalDownload int64
	x.Iterate(new(Package), func(idx int, bean interface{}) error {
		pkg := bean.(*Package)
		totalDownload += pkg.DownloadCount
		return nil
	})
	Statistic.NumTotalDownload = totalDownload
	Statistic.NumPackages, _ = x.Count(new(Package))
	Statistic.NumDownloaders, _ = x.Count(new(Downloader))

	Statistic.TrendingPackages = make([]*Package, 0, 15)
	x.Limit(15).Desc("recent_download").Find(&Statistic.TrendingPackages)

	Statistic.NewPackages = make([]*Package, 0, 15)
	x.Limit(15).Desc("created").Find(&Statistic.NewPackages)

	Statistic.PopularPackages = make([]*Package, 0, 15)
	x.Limit(15).Desc("download_count").Find(&Statistic.PopularPackages)
}

// uploadArchives checks and uploads local archives to QiNiu.
func uploadArchives() {
	revs, err := GetLocalRevisions()
	if err != nil {
		log.Error(4, "Fail to get local revisions: %v", err)
		return
	}

	// Upload.
	for _, rev := range revs {
		pkg, err := GetPakcageByID(rev.PkgID)
		if err != nil {
			log.Error(4, "Fail to get package by ID(%d): %v", rev.PkgID, err)
			continue
		}

		ext := archive.GetExtension(pkg.ImportPath)
		key := pkg.ImportPath + "-" + rev.Revision + ext
		localPath := path.Join(pkg.ImportPath, rev.Revision)
		fpath := path.Join(setting.ArchivePath, localPath+ext)

		// Move.
		// rsCli := rs.New(nil)
		// log.Info(key)
		// err = rsCli.Move(nil, setting.BucketName, pkg.ImportPath+"-"+rev.Revision, setting.BucketName, key)
		// if err != nil {
		// 	log.Error(4, rev.Revision)
		// }
		// continue

		if !com.IsFile(fpath) {
			log.Debug("Delete: %v", fpath)
			DeleteRevisionById(rev.ID)
			continue
		}

		// Check archive size.
		f, err := os.Open(fpath)
		if err != nil {
			log.Error(4, "Fail to open file(%s): %v", fpath, err)
			continue
		}
		fi, err := f.Stat()
		if err != nil {
			log.Error(4, "Fail to get file info(%s): %v", fpath, err)
			continue
		}
		// Greater then MAX_UPLOAD_SIZE.
		if fi.Size() > setting.MaxUploadSize<<20 {
			log.Debug("Ignore large archive: %v", fpath)
			continue
		}

		log.Debug("Uploading: %s", localPath)
		if err = qiniu.UploadArchive(key, fpath); err != nil {
			log.Error(4, "Fail to upload file(%s): %v", fpath, err)
			continue
		}
		rev.Storage = QINIU
		if err := UpdateRevision(rev); err != nil {
			log.Error(4, "Fail to upadte revision(%d): %v", rev.ID, err)
			continue
		}
		os.Remove(fpath)
		log.Info("Uploaded: %s", localPath)
	}
}
