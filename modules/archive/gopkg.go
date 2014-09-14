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
	"io/ioutil"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/Unknwon/com"

	"github.com/gpmgo/switch/modules/setting"
)

var (
	gopkgPathPattern = regexp.MustCompile(`^/(?:([a-zA-Z0-9][-a-zA-Z0-9]+)/)?([a-zA-Z][-.a-zA-Z0-9]*)\.((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2})(?:\.git)?((?:/[a-zA-Z0-9][-.a-zA-Z0-9]*)*)$`)
	gopkgPattern     = regexp.MustCompile(`^gopkg\.in`)
)

func getGopkgRevision(client *http.Client, n *Node) error {
	// Get real GitHub path.
	m := gopkgPathPattern.FindStringSubmatch(strings.TrimPrefix(n.ImportPath, "gopkg.in"))
	if m == nil {
		return fmt.Errorf("fail to match URL path")
	}
	user := m[1]
	name := m[2]
	if len(user) == 0 {
		user = "go-" + name
	}
	n.DownloadURL = path.Join("github.com", user, name)

	// Parse revision SHA by tag.
	req, err := http.NewRequest("GET", "https://"+n.DownloadURL+".git/info/refs?service=git-upload-pack", nil)
	if err != nil {
		return fmt.Errorf("fail to make refs request: %v", err)
	}
	req.Header.Set("User-Agent", com.UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fail to get response of refs: %v", err)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("fail to read response data of refs: %v", err)
	}
	branchRef := "refs/heads/" + m[3]
	tagRef := "refs/tags/" + m[3]
	lines := strings.Split(string(data), "\n")
	for _, line := range lines[2 : len(lines)-1] {
		if !strings.Contains(line, branchRef) && !strings.Contains(line, tagRef) {
			continue
		}
		n.Revision = line[4:44]
		break
	}

	if len(n.Revision) == 0 {
		return fmt.Errorf("cannot find revision in page: %s", n.ImportPath)
	}
	n.ArchivePath = path.Join(setting.ArchivePath, n.ImportPath, n.Revision+".zip")
	return nil
}

func getGopkgArchive(client *http.Client, match map[string]string, n *Node) error {
	// We use .zip here.
	// zip: https://github.com/{owner}/{repo}/archive/{sha}.zip

	// Downlaod archive.
	if err := com.HttpGetToFile(client,
		fmt.Sprintf("https://%s/archive/%s.zip", n.DownloadURL, n.Revision), nil, n.ArchivePath); err != nil {
		return fmt.Errorf("fail to download archive(%s): %v", n.ImportPath, err)
	}
	return nil
}
