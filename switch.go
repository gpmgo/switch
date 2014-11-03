// +build go1.2

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

// Switch is a server that provides versioning caching and delivering Go packages service.
package main

import (
	"fmt"
	"html/template"
	"net/http"
	"runtime"
	"strings"

	"github.com/Unknwon/macaron"
	"github.com/macaron-contrib/cache"
	"github.com/macaron-contrib/i18n"
	"github.com/macaron-contrib/session"
	"github.com/macaron-contrib/toolbox"

	"github.com/gpmgo/switch/models"
	"github.com/gpmgo/switch/modules/base"
	"github.com/gpmgo/switch/modules/log"
	"github.com/gpmgo/switch/modules/middleware"
	_ "github.com/gpmgo/switch/modules/qiniu"
	"github.com/gpmgo/switch/modules/setting"
	"github.com/gpmgo/switch/routers"
	"github.com/gpmgo/switch/routers/api/admin"
	"github.com/gpmgo/switch/routers/api/v1"
)

const APP_VER = "0.5.0.1102"

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	setting.AppVer = APP_VER
}

// newMacaron initializes Macaron instance.
func newMacaron() *macaron.Macaron {
	m := macaron.New()
	m.Use(macaron.Logger())
	m.Use(macaron.Recovery())
	m.Use(macaron.Static("public",
		macaron.StaticOptions{
			SkipLogging: !setting.DisableRouterLog,
		},
	))
	m.Use(macaron.Renderer(macaron.RenderOptions{
		Directory:  "templates",
		Funcs:      []template.FuncMap{base.TemplateFuncs},
		IndentJSON: macaron.Env != macaron.PROD,
	}))
	m.Use(i18n.I18n(i18n.Options{
		Langs:    setting.Langs,
		Names:    setting.Names,
		Redirect: true,
	}))
	m.Use(cache.Cacher(cache.Options{
		Adapter:  setting.CacheAdapter,
		Interval: setting.CacheInternal,
		Conn:     setting.CacheConn,
	}))
	m.Use(session.Sessioner(session.Options{
		Provider: setting.SessionProvider,
		Config:   *setting.SessionConfig,
	}))
	m.Use(toolbox.Toolboxer(m, toolbox.Options{
		HealthCheckFuncs: []*toolbox.HealthCheckFuncDesc{
			&toolbox.HealthCheckFuncDesc{
				Desc: "Database connection",
				Func: models.Ping,
			},
		},
	}))
	m.Use(middleware.Contexter())
	return m
}

func main() {
	log.Info("%s %s", setting.AppName, APP_VER)
	log.Info("Run Mode: %s", strings.Title(macaron.Env))
	setting.NewServices()

	m := newMacaron()

	// Routers.
	m.Get("/", routers.Home)
	m.Route("/download", "GET,POST", routers.Download)
	m.Get("/favicon.ico", func(ctx *middleware.Context) {
		ctx.Redirect("/img/favicon.png")
	})
	// m.Get("/search", routers.Search)
	// m.Get("/about", routers.About)

	// Documentation routers.
	m.Get("/docs/*", routers.Docs)

	// Package routers.
	m.Get("/*", routers.Package)
	m.Get("/badge/*", routers.Badge)

	// API routers.
	m.Group("/api", func() {
		m.Group("/v1", func() {
			m.Group("", func() {
				m.Get("/download", v1.Download)
				m.Get("/revision", v1.GetRevision)
			}, v1.PackageFilter())
		})

		// Admin APIs.
		m.Group("/admin", func() {
			m.Group("/package", func() {
				m.Post("/block", admin.BlockPackage)
				m.Get("/revision/large", admin.ListLargeRevisions)
			})
		}, admin.ValidateToken())
	})

	// Robots.txt
	m.Get("/robots.txt", func() string {
		return `User-agent: *
Disallow: /api/
Disallow: /download`
	})

	// Not found handler.
	m.NotFound(routers.NotFound)

	var err error
	listenAddr := fmt.Sprintf("%s:%s", setting.HttpAddr, setting.HttpPort)
	log.Info("Listen: %v://%s", setting.Protocol, listenAddr)
	switch setting.Protocol {
	case setting.HTTP:
		err = http.ListenAndServe(listenAddr, m)
	case setting.HTTPS:
		err = http.ListenAndServeTLS(listenAddr, setting.CertFile, setting.KeyFile, m)
	default:
		log.Fatal(4, "Invalid protocol: %s", setting.Protocol)
	}
	if err != nil {
		log.Fatal(4, "Fail to start server: %v", err)
	}
}
