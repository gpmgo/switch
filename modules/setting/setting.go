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
	"os"
	"path"
	"strings"

	"github.com/Unknwon/com"
	"github.com/Unknwon/goconfig"
	"github.com/Unknwon/macaron"
	"github.com/macaron-contrib/session"
	"github.com/qiniu/api/conf"

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
	ArchivePath        string
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

	// Session settings.
	SessionProvider string
	SessionConfig   *session.Config

	// Global setting objects.
	Cfg           *goconfig.ConfigFile
	ProdMode      bool
	RootPathPairs = map[string]int{
		"github.com":      3,
		"code.google.com": 3,
		"bitbucket.org":   3,
		"git.oschina.net": 3,
		"gitcafe.com":     3,
		"launchpad.net":   2,
	}
	ExtensionPairs = map[string]string{
		"github.com/":      ".zip",
		"code.google.com/": ".zip",
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

	if Cfg.MustValue("", "RUN_MODE", "dev") == "prod" {
		macaron.Env = macaron.PROD
		ProdMode = true
	}

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
	ArchivePath = Cfg.MustValue("server", "ARCHIVE_PATH", "data/archives")
	os.MkdirAll(ArchivePath, os.ModePerm)

	GithubCredentials = "client_id=" + Cfg.MustValue("github", "CLIENT_ID") +
		"&client_secret=" + Cfg.MustValue("github", "CLIENT_SECRET")

	BucketName = Cfg.MustValue("qiniu", "BUCKET_NAME")
	BucketUrl = Cfg.MustValue("qiniu", "BUCKET_URL")
	conf.ACCESS_KEY = Cfg.MustValue("qiniu", "ACCESS_KEY")
	conf.SECRET_KEY = Cfg.MustValue("qiniu", "SECRET_KEY")

	Langs = Cfg.MustValueArray("i18n", "LANGS", ",")
	Names = Cfg.MustValueArray("i18n", "NAMES", ",")
}

func newSessionService() {
	SessionProvider = Cfg.MustValueRange("session", "PROVIDER", "memory",
		[]string{"memory", "file", "redis", "mysql"})

	SessionConfig = new(session.Config)
	SessionConfig.ProviderConfig = strings.Trim(Cfg.MustValue("session", "PROVIDER_CONFIG"), "\" ")
	SessionConfig.CookieName = Cfg.MustValue("session", "COOKIE_NAME", "i_like_gogits")
	SessionConfig.Secure = Cfg.MustBool("session", "COOKIE_SECURE")
	SessionConfig.EnableSetCookie = Cfg.MustBool("session", "ENABLE_SET_COOKIE", true)
	SessionConfig.Gclifetime = Cfg.MustInt64("session", "GC_INTERVAL_TIME", 86400)
	SessionConfig.Maxlifetime = Cfg.MustInt64("session", "SESSION_LIFE_TIME", 86400)
	SessionConfig.SessionIDHashFunc = Cfg.MustValueRange("session", "SESSION_ID_HASHFUNC",
		"sha1", []string{"sha1", "sha256", "md5"})
	SessionConfig.SessionIDHashKey = Cfg.MustValue("session", "SESSION_ID_HASHKEY", string(com.RandomCreateBytes(16)))

	if SessionProvider == "file" {
		os.MkdirAll(path.Dir(SessionConfig.ProviderConfig), os.ModePerm)
	}

	log.Info("Session Service Enabled")
}

func NewServices() {
	newCacheService()
	newSessionService()
}
