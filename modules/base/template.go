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

package base

import (
	"container/list"
	"html/template"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Unknwon/i18n"
	"gopkg.in/fsnotify.v1"

	"github.com/gpmgo/switch/modules/log"
	"github.com/gpmgo/switch/modules/setting"
)

func monitorI18nLocale() {
	log.Trace("Monitor i18n locale files enabled")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(4, "Fail to init locale watcher: %v", err)
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				switch filepath.Ext(event.Name) {
				case ".ini":
					if err := i18n.ReloadLangs(); err != nil {
						log.Error(4, "Fail to relaod locale file reloaded: %v", err)
					}
					log.Trace("Locale file reloaded: %s", strings.TrimPrefix(event.Name, "conf/locale/"))
				}
			}
		}
	}()

	if err := watcher.Add("conf/locale"); err != nil {
		log.Fatal(4, "Fail to start locale watcher: %v", err)
	}
}

func init() {
	if !setting.ProdMode {
		monitorI18nLocale()
	}
}

func Str2html(raw string) template.HTML {
	return template.HTML(raw)
}

func Range(l int) []int {
	return make([]int, l)
}

func List(l *list.List) chan interface{} {
	e := l.Front()
	c := make(chan interface{})
	go func() {
		for e != nil {
			c <- e.Value
			e = e.Next()
		}
		close(c)
	}()
	return c
}

func ShortSha(sha1 string) string {
	if len(sha1) == 40 {
		return sha1[:10]
	}
	return sha1
}

var mailDomains = map[string]string{
	"gmail.com": "gmail.com",
}

var TemplateFuncs template.FuncMap = map[string]interface{}{
	"GoVer": func() string {
		return runtime.Version()
	},
	"AppName": func() string {
		return setting.AppName
	},
	"AppVer": func() string {
		return setting.AppVer
	},
	"AppDomain": func() string {
		return setting.Domain
	},
	"AvatarLink": AvatarLink,
	"str2html":   Str2html,
	"TimeSince":  TimeSince,
	"FileSize":   FileSize,
	"Subtract":   Subtract,
	"Add": func(a, b int) int {
		return a + b
	},
	"DateFormat": DateFormat,
	"List":       List,
	"Mail2Domain": func(mail string) string {
		if !strings.Contains(mail, "@") {
			return "try.gogits.org"
		}

		suffix := strings.SplitN(mail, "@", 2)[1]
		domain, ok := mailDomains[suffix]
		if !ok {
			return "mail." + suffix
		}
		return domain
	},
	"SubStr": func(str string, start, length int) string {
		return str[start : start+length]
	},
	"ShortSha": ShortSha,
	"i18n":     i18n.Tr,
}
