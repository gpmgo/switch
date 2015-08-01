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
	"net/http"
	"runtime"
	"strings"

	"github.com/Unknwon/macaron"
	"github.com/macaron-contrib/i18n"
	"github.com/macaron-contrib/pongo2"
	"github.com/macaron-contrib/session"

	"github.com/gpmgo/switch/modules/log"
	"github.com/gpmgo/switch/modules/middleware"
	_ "github.com/gpmgo/switch/modules/qiniu"
	"github.com/gpmgo/switch/modules/setting"
	"github.com/gpmgo/switch/routers"
	"github.com/gpmgo/switch/routers/admin"
	"github.com/gpmgo/switch/routers/api/v1"
)

const APP_VER = "0.6.5.0801"

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	setting.AppVer = APP_VER
}

func main() {
	log.Info("%s %s", setting.AppName, APP_VER)
	log.Info("Run Mode: %s", strings.Title(macaron.Env))

	m := macaron.New()
	m.Use(macaron.Logger())
	m.Use(macaron.Recovery())
	m.Use(macaron.Static("public", macaron.StaticOptions{
		SkipLogging: true,
	}))
	m.Use(pongo2.Pongoers(pongo2.Options{
		Directory:  "templates/web",
		IndentJSON: macaron.Env != macaron.PROD,
	}, "templates/admin"))
	m.Use(i18n.I18n())
	m.Use(session.Sessioner())
	m.Use(middleware.Contexter())

	// Routes.
	m.Get("/", routers.Home)
	m.Route("/download", "GET,POST", routers.Download)
	m.Get("/favicon.ico", func(ctx *middleware.Context) {
		ctx.Redirect("/img/favicon.png")
	})
	// m.Get("/search", routers.Search)
	// m.Get("/about", routers.About)

	// Package.
	m.Get("/*", routers.Package)
	m.Get("/badge/*", routers.Badge)

	// Admin.
	m.Post("/admin/auth", admin.AuthPost)
	m.Group("/admin", func() {
		m.Get("", admin.Dashboard)

		m.Group("/packages", func() {
			m.Get("", admin.Revisions)
			m.Get("/larges", admin.LargeRevisions)
		})

		m.Group("/blocks", func() {
			m.Get("", admin.Blocks)
			m.Combo("/new").Get(admin.BlockPackage).Post(admin.BlockPackagePost)
			m.Get("/:id:int/delete", admin.UnblockPackage)

			m.Group("/rules", func() {
				m.Get("", admin.BlockRules)
				m.Combo("/new").Get(admin.NewBlockRule).Post(admin.NewBlockRulePost)
				m.Get("/:id:int/run", admin.RunRule)
				m.Get("/:id:int/delete", admin.DeleteBlockRule)
			})
		})
	}, admin.Auth)

	// API.
	m.Group("/api", func() {
		m.Group("/v1", func() {
			m.Group("", func() {
				m.Get("/download", v1.Download)
				m.Get("/revision", v1.GetRevision)
			}, v1.PackageFilter())
		})
	})

	// Robots.txt
	m.Get("/robots.txt", func() string {
		return `User-agent: *
Disallow: /api/
Disallow: /download`
	})

	m.NotFound(routers.NotFound)

	listenAddr := fmt.Sprintf("0.0.0.0:%d", setting.HttpPort)
	log.Info("Listen: http://%s", listenAddr)
	if err := http.ListenAndServe(listenAddr, m); err != nil {
		log.Fatal(4, "Fail to start server: %v", err)
	}
}
