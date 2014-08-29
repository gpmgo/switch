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

package setting

import (
	"strings"

	"github.com/Unknwon/com"
	"github.com/Unknwon/goconfig"

	"github.com/gpmgo/switch/modules/log"
)

type Scheme string

const (
	HTTP  Scheme = "http"
	HTTPS Scheme = "https"
)

var (
	// App settings.
	AppVer  string
	AppName string
	AppLogo string
	AppUrl  string

	// Server settings.
	Protocol           Scheme
	Domain             string
	HttpAddr, HttpPort string
	DisableRouterLog   bool
	CertFile, KeyFile  string
	ArchivePath        = "data/archives"
	EnableGzip         bool

	// Security settings.
	SecretKey          = "!#@FDEWREWR&*("
	LogInRememberDays  = 7
	CookieUserName     = "gopm_awesome"
	CookieRememberName = "gopm_incredible"

	// Cache settings.
	CacheAdapter  string
	CacheInternal int
	CacheConn     string

	EnableRedis    bool
	EnableMemcache bool

	// Global setting objects.
	Cfg        *goconfig.ConfigFile
	ProdMode   bool
	Languages  []string
	Categories []string = []string{
		"Web Frameworks",
		"Configuration File Parsers",
		"Logging",
		"Databases and Storage",
	}
	GoVersions []string = []string{
		"Go 1.1",
		"Go 1.2",
		"Go 1.3",
	}

	// I18n settings.
	Langs, Names []string
)

var Service struct {
	RegisterEmailConfirm bool
	ActiveCodeLives      int
	ResetPwdCodeLives    int
}

func newCacheService() {
	CacheAdapter = Cfg.MustValueRange("cache", "ADAPTER", "memory", []string{"memory", "redis", "memcache"})
	if EnableRedis {
		log.Info("Redis Enabled")
	}
	if EnableMemcache {
		log.Info("Memcache Enabled")
	}

	switch CacheAdapter {
	case "memory":
		CacheInternal = Cfg.MustInt("cache", "INTERVAL", 60)
	case "redis", "memcache":
		CacheConn = strings.Trim(Cfg.MustValue("cache", "HOST"), "\" ")
	default:
		log.Fatal(4, "Unknown cache adapter: %s", CacheAdapter)
	}

	log.Info("Cache Service Enabled")
}

func init() {
	log.NewLogger(0, "console", `{"level": 0}`)

	var err error
	Cfg, err = goconfig.LoadConfigFile("conf/app.ini")
	if err != nil {
		log.Fatal(4, "Fail to parse 'conf/app.ini': %v", err)
	}
	if com.IsFile("custom/app.ini") {
		if err = Cfg.AppendFiles("custom/app.ini"); err != nil {
			log.Fatal(4, "Fail to load 'custom/app.ini': %v", err)
		}
	}

	AppName = Cfg.MustValue("", "APP_NAME")
	AppLogo = Cfg.MustValue("", "APP_LOGO", "img/favicon.png")
	AppUrl = Cfg.MustValue("server", "ROOT_URL", "http://localhost:8084")

	Protocol = HTTP
	if Cfg.MustValue("server", "PROTOCOL") == "https" {
		Protocol = HTTPS
		CertFile = Cfg.MustValue("server", "CERT_FILE")
		KeyFile = Cfg.MustValue("server", "KEY_FILE")
	}
	Domain = Cfg.MustValue("server", "DOMAIN", "localhost")
	HttpAddr = Cfg.MustValue("server", "HTTP_ADDR", "0.0.0.0")
	HttpPort = Cfg.MustValue("server", "HTTP_PORT", "8084")
	DisableRouterLog = Cfg.MustBool("server", "DISABLE_ROUTER_LOG")

	Langs = Cfg.MustValueArray("i18n", "LANGS", ",")
	Names = Cfg.MustValueArray("i18n", "NAMES", ",")
}

func NewServices() {
	newCacheService()
}
