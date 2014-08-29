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

package routers

import (
	"github.com/gpmgo/switch/models"
	"github.com/gpmgo/switch/modules/middleware"
)

func Home(ctx *middleware.Context) {
	ctx.Data["Title"] = ctx.Tr("app_title")
	ctx.Data["PageIsHome"] = true
	ctx.Data["Stats"] = models.Statistic()
	ctx.HTML(200, "home")
}

func Search(ctx *middleware.Context) {
	ctx.Data["Title"] = ctx.Locale.Tr("search")
	ctx.Data["SearchKeyword"] = ctx.Query("q")

	pkgs, err := models.SearchPackages(ctx.Query("q"))
	if err != nil {
		ctx.Handle(500, "SearchPackages", err)
		return
	}
	ctx.Data["ResultPackages"] = pkgs
	ctx.HTML(200, "search")
}

func About(ctx *middleware.Context) {
	ctx.Data["Title"] = ctx.Locale.Tr("about_lower")
	ctx.Data["PageIsAbout"] = true
	ctx.HTML(200, "about")
}

func NotFound(ctx *middleware.Context) {
	ctx.Data["Title"] = ctx.Tr("status.page_not_found")
	ctx.Handle(404, "home.NotFound", nil)
}
