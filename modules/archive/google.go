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

package archive

import (
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/Unknwon/com"

	"github.com/gpmgo/switch/modules/setting"
)

var (
	googleRevisionPattern = regexp.MustCompile(`_setViewedRevision\('[a-z0-9A-Z]+`)
	googleRepoRe          = regexp.MustCompile(`id="checkoutcmd">(hg|git|svn)`)
	googlePattern         = regexp.MustCompile(`^code\.google\.com/p/(?P<repo>[a-z0-9\-]+)(:?\.(?P<subrepo>[a-z0-9\-]+))?(?P<dir>/[a-z0-9A-Z_.\-/]+)?$`)
)

func setupGoogleMatch(match map[string]string) {
	if s := match["subrepo"]; s != "" {
		match["dot"] = "."
		match["query"] = "?repo=" + s
	} else {
		match["dot"] = ""
		match["query"] = ""
	}
}

func getGoogleVCS(client *http.Client, match map[string]string) error {
	// Scrape the HTML project page to find the VCS.
	p, err := com.HttpGetBytes(client, com.Expand("http://code.google.com/p/{repo}/source/checkout", match), nil)
	if err != nil {
		return fmt.Errorf("fail to fetch page: %v", err)
	}
	m := googleRepoRe.FindSubmatch(p)
	if m == nil {
		return com.NotFoundError{"Could not VCS on Google Code project page."}
	}
	match["vcs"] = string(m[1])
	return nil
}

func getGoogleRevision(client *http.Client, n *Node) error {
	match := map[string]string{}
	{
		m := googlePattern.FindStringSubmatch(n.ImportPath)
		for i, n := range googlePattern.SubexpNames() {
			if n != "" {
				match[n] = m[i]
			}
		}
		setupGoogleMatch(match)
	}

	if len(n.Value) == 0 {
		// Scrape the HTML project page to find the VCS.
		p, err := com.HttpGetBytes(client, com.Expand("http://code.google.com/p/{repo}/source/checkout", match), nil)
		if err != nil {
			return fmt.Errorf("fail to fetch page: %v", err)
		}
		m := googleRepoRe.FindSubmatch(p)
		if m == nil {
			return fmt.Errorf("cannot find VCS on Google Code project page")
		}
		match["vcs"] = string(m[1])
		n.Value = defaultTags[match["vcs"]]
	}
	match["tag"] = n.Value
	data, err := com.HttpGetBytes(client, com.Expand("http://code.google.com/p/{repo}/source/browse/?repo={subrepo}&r={tag}", match), nil)
	if err != nil {
		return fmt.Errorf("fail to get revision(%s): %v", n.ImportPath, err)
	}
	m := googleRevisionPattern.FindSubmatch(data)
	if m == nil {
		return fmt.Errorf("cannot find revision in page: %s", n.ImportPath)
	}
	n.Revision = strings.TrimPrefix(string(m[0]), `_setViewedRevision('`)
	n.ArchivePath = path.Join(setting.ArchivePath, n.ImportPath, n.Revision+".zip")
	return nil
}

func getGoogleArchive(client *http.Client, match map[string]string, n *Node) error {
	setupGoogleMatch(match)
	match["tag"] = n.Revision

	if match["vcs"] == "svn" {
		return fmt.Errorf("SVN not support yet")
	} else {
		// Downlaod archive.
		if err := com.HttpGetToFile(client,
			com.Expand("http://{subrepo}{dot}{repo}.googlecode.com/archive/{tag}.zip", match), nil, n.ArchivePath); err != nil {
			return fmt.Errorf("fail to download archive: %s", n.ImportPath, err)
		}
	}
	return nil
}
