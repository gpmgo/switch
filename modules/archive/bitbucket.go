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
	bitbucketRevisionPattern = regexp.MustCompile(`data-revision="[a-z0-9A-Z]+`)
	bitbucketPattern         = regexp.MustCompile(`^bitbucket\.org/(?P<owner>[a-z0-9A-Z_.\-]+)/(?P<repo>[a-z0-9A-Z_.\-]+)(?P<dir>/[a-z0-9A-Z_.\-/]*)?$`)
	bitbucketEtagRe          = regexp.MustCompile(`^(hg|git)-`)
)

func getBitbucketRevision(client *http.Client, n *Node) error {
	if len(n.Value) == 0 {
		var repo struct {
			Scm string
		}
		if err := com.HttpGetJSON(client, fmt.Sprintf("https://api.bitbucket.org/1.0/repositories/%s", strings.TrimPrefix(n.ImportPath, "bitbucket.org/")), &repo); err != nil {
			return fmt.Errorf("fail to fetch page: %v", err)
		}
		n.Value = defaultTags[repo.Scm]
	}
	data, err := com.HttpGetBytes(client, fmt.Sprintf("https://%s/commits/%s", n.ImportPath, n.Value), nil)
	if err != nil {
		return fmt.Errorf("fail to get revision(%s): %v", n.ImportPath, err)
	}
	m := bitbucketRevisionPattern.FindSubmatch(data)
	if m == nil {
		return fmt.Errorf("cannot find revision in page: %s", n.ImportPath)
	}
	n.Revision = strings.TrimPrefix(string(m[0]), `data-revision="`)
	n.ArchivePath = path.Join(setting.ArchivePath, n.ImportPath, n.Revision+".zip")
	return nil
}

func getBitbucketArchive(client *http.Client, match map[string]string, n *Node) error {
	match["sha"] = n.Revision

	// Downlaod archive.
	if err := com.HttpGetToFile(client,
		com.Expand("https://bitbucket.org/{owner}/{repo}/get/{sha}.zip", match), nil, n.ArchivePath); err != nil {
		return fmt.Errorf("fail to download archive: %s", n.ImportPath, err)
	}
	return nil
}
