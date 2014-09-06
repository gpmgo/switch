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
	githubRevisionPattern = regexp.MustCompile(`js-selectable-text">[a-z0-9A-Z]+`)
	githubPattern         = regexp.MustCompile(`^github\.com/(?P<owner>[a-z0-9A-Z_.\-]+)/(?P<repo>[a-z0-9A-Z_.\-]+)(?P<dir>/[a-z0-9A-Z_.\-/]*)?$`)
)

func getGithubRevision(client *http.Client, n *Node) error {
	if len(n.Value) == 0 {
		n.Value = "master"
	}
	data, err := com.HttpGetBytes(client, fmt.Sprintf("https://%s/commit/%s", n.ImportPath, n.Value), nil)
	if err != nil {
		return fmt.Errorf("fail to get revision(%s): %v", n.ImportPath, err)
	}
	m := githubRevisionPattern.FindSubmatch(data)
	if m == nil {
		return fmt.Errorf("cannot find revision in page: %s", n.ImportPath)
	}
	n.Revision = strings.TrimPrefix(string(m[0]), `js-selectable-text">`)
	n.ArchivePath = path.Join(setting.ArchivePath, n.ImportPath, n.Revision+".zip")
	return nil
}

func getGithubArchive(client *http.Client, match map[string]string, n *Node) error {
	match["sha"] = n.Revision
	// match["cred"] = setting.GithubCredentials

	// We use .zip here.
	// zip: https://github.com/{owner}/{repo}/archive/{sha}.zip
	// tarball: https://github.com/{owner}/{repo}/tarball/{sha}

	// Downlaod archive.
	if err := com.HttpGetToFile(client,
		com.Expand("https://github.com/{owner}/{repo}/archive/{sha}.zip", match), nil, n.ArchivePath); err != nil {
		return fmt.Errorf("fail to download archive: %s", n.ImportPath, err)
	}
	return nil
}
