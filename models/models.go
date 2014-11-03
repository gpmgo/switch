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

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-xorm/xorm"
	"github.com/robfig/cron"

	"github.com/gpmgo/switch/modules/log"
	"github.com/gpmgo/switch/modules/setting"
)

var (
	x *xorm.Engine
)

func init() {
	var err error
	x, err = xorm.NewEngine("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8",
		setting.Cfg.MustValue("database", "USER"),
		setting.Cfg.MustValue("database", "PASSWD"),
		setting.Cfg.MustValue("database", "HOST"),
		setting.Cfg.MustValue("database", "NAME")))
	if err != nil {
		log.Fatal(4, "Fail to init new engine: %v", err)
	} else if err = x.Sync(new(Package), new(Revision), new(Downloader), new(Block)); err != nil {
		log.Fatal(4, "Fail to sync database: %v", err)
	}

	statistic()
	c := cron.New()
	c.AddFunc("@every 5m", statistic)
	c.Start()
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
