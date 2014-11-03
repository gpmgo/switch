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
	"github.com/Unknwon/macaron"

	"github.com/gpmgo/switch/modules/middleware"
	"github.com/gpmgo/switch/modules/setting"
)

func ValidateToken() macaron.Handler {
	return func(ctx *middleware.Context) {
		if len(setting.AccessToken) == 0 {
			ctx.JSON(500, map[string]string{
				"error": "no access token configurated",
			})
			return
		}
		if ctx.Query("access_token") != setting.AccessToken {
			ctx.JSON(500, map[string]string{
				"error": "access denied",
			})
			return
		}
	}
}
