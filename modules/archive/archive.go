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
	"errors"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/gpmgo/switch/modules/setting"
)

var (
	ErrNotMatchAnyService = errors.New("cannot match any service")
)

// A Node represents a node object to be fetched from remote.
type Node struct {
	ImportPath  string // Package root import path.
	DownloadURL string // Actual download URL can be different from import path.
	Value       string
	Revision    string
	ArchivePath string
}

func joinPath(name string, num int) string {
	subdirs := strings.Split(name, "/")
	if len(subdirs) > num {
		return strings.Join(subdirs[:num], "/")
	}
	return name
}

// GetRootPath returns project root path.
func GetRootPath(name string) string {
	for prefix, num := range setting.RootPathPairs {
		if strings.HasPrefix(name, prefix) {
			return joinPath(name, num)
		}
	}

	if strings.HasPrefix(name, "gopkg.in") {
		m := gopkgPathPattern.FindStringSubmatch(strings.TrimPrefix(name, "gopkg.in"))
		if m == nil {
			return name
		}
		user := m[1]
		repo := m[2]
		if len(user) == 0 {
			user = "go-" + repo
		}
		return path.Join("gopkg.in", user, repo+m[3])
	}
	return name
}

// GetExtension returns extension by import path.
func GetExtension(importPath string) string {
	for prefix, ext := range setting.ExtensionPairs {
		if strings.HasPrefix(importPath, prefix) {
			return ext
		}
	}
	return ".zip"
}

// NewNode initializes and returns a new Node representation.
func NewNode(importPath string, val string) *Node {
	return &Node{
		ImportPath:  GetRootPath(importPath),
		Value:       val,
		DownloadURL: GetRootPath(importPath),
	}
}

type (
	// service represents a source code control service.
	service struct {
		pattern *regexp.Regexp
		prefix  string
		get     func(*http.Client, map[string]string, *Node) error
	}
	revService struct {
		prefix string
		get    func(*http.Client, *Node) error
	}
)

var (
	// services is the list of source code control services handled by gopm.
	services = []*service{
		{githubPattern, "github.com/", getGithubArchive},
		{googlePattern, "code.google.com/", getGoogleArchive},
		{bitbucketPattern, "bitbucket.org/", getBitbucketArchive},
		// {oscPattern, "git.oschina.net/", getOscPkg},
		// {gitcafePattern, "gitcafe.com/", getGitcafePkg},
		// {launchpadPattern, "launchpad.net/", getLaunchpadPkg},
		{gopkgPattern, "gopkg.in/", getGopkgArchive},
	}
	revServices = []*revService{
		{"github.com/", getGithubRevision},
		{"code.google.com/", getGoogleRevision},
		{"bitbucket.org/", getBitbucketRevision},
		{"gopkg.in/", getGopkgRevision},
	}
	defaultTags = map[string]string{"git": "master", "hg": "default", "svn": "trunk"}
)

// GetRevision fetches revision of node from service.
func (n *Node) GetRevision() error {
	for _, s := range revServices {
		if !strings.HasPrefix(n.ImportPath, s.prefix) {
			continue
		}
		return s.get(HttpClient, n)
	}
	return ErrNotMatchAnyService
}

// Download downloads remote package without version control.
func (n *Node) Download() error {
	for _, s := range services {
		if !strings.HasPrefix(n.DownloadURL, s.prefix) {
			continue
		}

		m := s.pattern.FindStringSubmatch(n.DownloadURL)
		if m == nil {
			if s.prefix != "" {
				return errors.New("Cannot match package service prefix by given path")
			}
			continue
		}

		match := map[string]string{"downloadURL": n.DownloadURL}
		for i, n := range s.pattern.SubexpNames() {
			if n != "" {
				match[n] = m[i]
			}
		}
		return s.get(HttpClient, match, n)
	}

	if n.ImportPath != n.DownloadURL {
		return errors.New("Didn't find any match service")
	}
	return ErrNotMatchAnyService
	// log.Log("Cannot match any service, getting dynamic...")
	// return n.getDynamic(HttpClient, ctx)
}
