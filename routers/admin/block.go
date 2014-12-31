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

	"github.com/gpmgo/switch/models"
	"github.com/gpmgo/switch/modules/log"
	"github.com/gpmgo/switch/modules/middleware"
	"github.com/gpmgo/switch/modules/qiniu"
	"github.com/gpmgo/switch/modules/setting"
)

func Blocks(ctx *middleware.Context) {
	ctx.Data["PageIsBlocks"] = true
	ctx.Data["PageIsBlocksList"] = true
	ctx.HTML(200, "blocks/list")
}

func BlockRules(ctx *middleware.Context) {
	ctx.Data["PageIsBlocks"] = true
	ctx.Data["PageIsBlocksRules"] = true

	rules, err := models.ListBlockRules(0)
	if err != nil {
		ctx.Handle(500, "ListBlockRules", err)
		return
	}
	ctx.Data["Rules"] = rules

	ctx.HTML(200, "blocks/rules")
}

func NewBlockRule(ctx *middleware.Context) {
	ctx.Data["PageIsBlocks"] = true
	ctx.Data["PageIsBlocksRules"] = true
	ctx.HTML(200, "blocks/rules_new")
}

func NewBlockRulePost(ctx *middleware.Context) {
	ctx.Data["PageIsBlocks"] = true
	ctx.Data["PageIsBlocksRules"] = true

	r := &models.BlockRule{
		Rule: ctx.Query("rule"),
		Note: ctx.Query("note"),
	}
	if err := models.NewBlockRule(r); err != nil {
		ctx.Handle(500, "NewBlockRule", err)
		return
	}

	ctx.Flash.Success("New block rule has been added!")
	ctx.Redirect("/admin/blocks/rules")
}

func RunRule(ctx *middleware.Context) {
	ctx.Data["PageIsBlocks"] = true
	ctx.Data["PageIsBlocksRules"] = true

	rid := ctx.ParamsInt64(":id")
	count, keys, err := models.RunBlockRule(rid)
	if err != nil {
		ctx.Handle(500, "RunBlockRule", err)
		return
	}

	if setting.ProdMode {
		for _, k := range keys {
			log.Trace("Deleting archive: %s", k)
			if err = qiniu.DeleteArchive(k); err != nil {
				log.Error(4, "Fail to delete archive(%s): %v", k, err)
			}
			log.Info("Archive deleted: %s", k)
		}
	}

	ctx.Flash.Success(fmt.Sprintf("%d packages are blocked by rule ID: %d.", count, rid))
	ctx.Redirect("/admin/blocks/rules")
}

func DeleteBlockRule(ctx *middleware.Context) {
	ctx.Data["PageIsBlocks"] = true
	ctx.Data["PageIsBlocksRules"] = true

	if err := models.DeleteBlockRule(ctx.ParamsInt64(":id")); err != nil {
		ctx.Handle(500, "DeleteBlockRule", err)
		return
	}

	ctx.Flash.Success("Block rule has been deleted!")
	ctx.Redirect("/admin/blocks/rules")
}
