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

package setting

import (
	"os"

	"github.com/Unknwon/com"
	"github.com/Unknwon/macaron"
	"github.com/qiniu/api/conf"
	"gopkg.in/ini.v0"

	"github.com/gpmgo/switch/modules/log"
)

var (
	// App settings.
	AppVer  string
	AppName string

	// Server settings.
	HttpPort      int
	ArchivePath   string
	MaxUploadSize int64

	// Security settings.
	SecretKey          = "!#@FDEWREWR&*("
	LogInRememberDays  = 7
	CookieUserName     = "gopm_awesome"
	CookieRememberName = "gopm_incredible"

	// Admin settings.
	AccessToken string

	// Global setting objects.
	Cfg           *ini.File
	ProdMode      bool
	RootPathPairs = map[string]int{
		"github.com":      3,
		"code.google.com": 3,
		"bitbucket.org":   3,
		"git.oschina.net": 3,
		"gitcafe.com":     3,
		"launchpad.net":   2,
		"golang.org":      3,
	}
	ExtensionPairs = map[string]string{
		"github.com/":      ".zip",
		"code.google.com/": ".zip",
		"golang.org/":      ".zip",
		"bitbucket.org/":   ".zip",
		"git.oschina.net/": ".zip",
		"gitcafe.com/":     ".tar",
		"gopkg.in/":        ".zip",
		"launchpad.net/":   ".tar.gz",
	}
	GithubCredentials string

	// QiNiu settings.
	BucketName string
	BucketUrl  string
)

var Service struct {
	RegisterEmailConfirm bool
	ActiveCodeLives      int
	ResetPwdCodeLives    int
}

func init() {
	log.NewLogger(0, "console", `{"level": 0}`)

	sources := []interface{}{"conf/app.ini"}
	if com.IsFile("custom/app.ini") {
		sources = append(sources, "custom/app.ini")
	}
	if err := macaron.SetConfig(sources[0], sources[1:]...); err != nil {
		log.Fatal(4, "Fail to set configuration: %v", err)
	}

	Cfg = macaron.Config()

	AppName = Cfg.Section("").Key("APP_NAME").String()

	if Cfg.Section("").Key("RUN_MODE").MustString("dev") == "prod" {
		macaron.Env = macaron.PROD
		ProdMode = true
	}

	HttpPort = Cfg.Section("server").Key("HTTP_PORT").MustInt(8084)
	ArchivePath = Cfg.Section("server").Key("ARCHIVE_PATH").MustString("data/archives")
	os.MkdirAll(ArchivePath, os.ModePerm)

	MaxUploadSize = Cfg.Section("server").Key("MAX_UPLOAD_SIZE").MustInt64(5)

	GithubCredentials = "client_id=" + Cfg.Section("github").Key("CLIENT_ID").String() +
		"&client_secret=" + Cfg.Section("github").Key("CLIENT_SECRET").String()

	conf.UP_HOST = Cfg.Section("qiniu").Key("UP_HOST").MustString(conf.UP_HOST)
	BucketName = Cfg.Section("qiniu").Key("BUCKET_NAME").String()
	BucketUrl = Cfg.Section("qiniu").Key("BUCKET_URL").String()
	conf.ACCESS_KEY = Cfg.Section("qiniu").Key("ACCESS_KEY").String()
	conf.SECRET_KEY = Cfg.Section("qiniu").Key("SECRET_KEY").String()

	AccessToken = Cfg.Section("admin").Key("ACCESS_TOKEN").String()
}
