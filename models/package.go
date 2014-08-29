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

package models

import (
	"errors"
	"strings"
	"time"

	"github.com/gpmgo/switch/modules/base"
	"github.com/gpmgo/switch/modules/log"
)

var (
	ErrPackageAlreadyExist = errors.New("Package already exist")
	ErrPackageNotExist     = errors.New("Package does not exist")
	ErrGoVersionNotExist   = errors.New("Go version does not exist")
	ErrCategoryNotExist    = errors.New("Category does not exist")
	ErrReleaseNotExist     = errors.New("Release does not exist")
)

type PkgSource int

const (
	REGISTRY PkgSource = iota + 1
)

type GoVersion int

const (
	GO11 GoVersion = iota + 1
	GO12
	GO13
)

type Release struct {
	Id            int64     `json:"id"`
	PkgId         int64     `xorm:"UNIQUE(s) INDEX" json:"pkg_id"`
	Tag           string    `xorm:"UNIQUE(s)" json:"tag"`
	Source        PkgSource `json:"-"`
	GoVer         GoVersion `xorm:"INDEX" json:"go_ver"`
	GoVerName     string    `xorm:"-" json:"-"`
	DownloadCount int64     `json:"download_count"`
	IsUploaded    bool      `json:"-"` // Indicates whether uploaded to QiNiu or not.
	Uploaded      time.Time `json:"-"`
	Created       time.Time `xorm:"CREATED" json:"created"`
}

func (r *Release) SetGoVersion(name string) error {
	switch name {
	case "Go 1.1":
		r.GoVer = GO11
	case "Go 1.2":
		r.GoVer = GO12
	case "Go 1.3":
		r.GoVer = GO13
	default:
		return ErrGoVersionNotExist
	}
	return nil
}

func (r *Release) GetGoVersion() {
	switch r.GoVer {
	case GO11:
		r.GoVerName = "Go 1.1"
	case GO12:
		r.GoVerName = "Go 1.2"
	case GO13:
		r.GoVerName = "Go 1.3"
	}
}

// NewRelease creates record of a new release.
func NewRelease(pkg *Package, r *Release) (err error) {
	sess := x.NewSession()
	defer sess.Close()
	if err = sess.Begin(); err != nil {
		return err
	}

	if _, err = sess.Insert(r); err != nil {
		sess.Rollback()
		return err
	}
	pkg.ReleaseIds = base.ToStr(r.Id) + "|" + pkg.ReleaseIds
	if _, err = sess.Id(pkg.Id).Update(pkg); err != nil {
		sess.Rollback()
		return err
	}
	return sess.Commit()
}

// GetReleaseById returns the release by given ID if exists.
func GetReleaseById(id int64) (*Release, error) {
	r := new(Release)
	has, err := x.Id(id).Get(r)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrReleaseNotExist
	}
	return r, nil
}

type PkgCategory int

const (
	WEB_FRAMEWORK PkgCategory = iota + 1
	CONFIGURATION
	LOGGING
	DATABASE
)

type Package struct {
	Id            int64
	OwnerId       int64  `xorm:"UNIQUE(s) INDEX"`
	Name          string `xorm:"UNIQUE(s)"`
	FullName      string
	Description   string
	Category      PkgCategory `xorm:"INDEX"`
	CategoryName  string      `xorm:"-"`
	ReleaseIds    string      `xorm:"TEXT"`
	Releases      []*Release  `xorm:"-"`
	Homepage      string
	Issues        string
	Donate        string
	DownloadCount int64
	IsPrivate     bool
	Created       time.Time `xorm:"CREATED"`
	Updated       time.Time `xorm:"UPDATED"`
}

func (pkg *Package) SetCategory(name string) error {
	switch name {
	case "Web Frameworks":
		pkg.Category = WEB_FRAMEWORK
	case "Configuration File Parsers":
		pkg.Category = CONFIGURATION
	case "Logging":
		pkg.Category = LOGGING
	case "Databases and Storage":
		pkg.Category = DATABASE
	default:
		return ErrCategoryNotExist
	}
	return nil
}

func (pkg *Package) GetCategory() {
	switch pkg.Category {
	case WEB_FRAMEWORK:
		pkg.CategoryName = "Web Frameworks"
	case CONFIGURATION:
		pkg.CategoryName = "Configuration File Parsers"
	case LOGGING:
		pkg.CategoryName = "Logging"
	case DATABASE:
		pkg.CategoryName = "Databases and Storage"
	}
}

func (pkg *Package) GetReleases() {
	ids := strings.Split(pkg.ReleaseIds, "|")
	pkg.Releases = make([]*Release, 0, len(ids))
	for _, idStr := range ids {
		if len(idStr) == 0 {
			continue
		}
		id, _ := base.StrTo(idStr).Int64()
		if id > 0 {
			r, err := GetReleaseById(id)
			if err != nil {
				log.Error(4, "Package.GetReleases(GetReleaseById): %v", err)
				continue
			}
			pkg.Releases = append(pkg.Releases, r)
		}
	}
}

// IsPackageNameUsed returns true if package name has been used of given user.
func IsPackageNameUsed(uid int64, name string) (bool, error) {
	if uid <= 0 || len(name) == 0 {
		return false, nil
	}
	return x.Get(&Package{OwnerId: uid, Name: strings.ToLower(name)})
}

// NewPackage creates record of a new package.
func NewPackage(pkg *Package) error {
	isExist, err := IsPackageNameUsed(pkg.OwnerId, pkg.Name)
	if err != nil {
		return err
	} else if isExist {
		return ErrPackageAlreadyExist
	}

	sess := x.NewSession()
	defer sess.Close()
	if err = sess.Begin(); err != nil {
		return err
	}

	pkg.Name = strings.ToLower(pkg.Name)
	if _, err = sess.Insert(pkg); err != nil {
		sess.Rollback()
		return err
	}

	rawSql := "UPDATE `user` SET num_packages = num_packages + 1 WHERE id = ?"
	if _, err = sess.Exec(rawSql, pkg.OwnerId); err != nil {
		sess.Rollback()
		return err
	}
	return sess.Commit()
}

// GetPackageByName returns package by given user and name.
func GetPackageByName(uid int64, name string) (*Package, error) {
	if uid <= 0 || len(name) == 0 {
		return nil, ErrPackageNotExist
	}

	pkg := &Package{OwnerId: uid, Name: name}
	has, err := x.Get(pkg)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrPackageNotExist
	}
	return pkg, nil
}

// SearchPackages searchs packages by given keyword.
func SearchPackages(keys string) ([]*Package, error) {
	keys = strings.TrimSpace(keys)
	if len(keys) == 0 {
		return nil, nil
	}
	key := strings.Split(keys, " ")[0]
	if len(key) == 0 {
		return nil, nil
	}

	pkgs := make([]*Package, 0, 50)
	err := x.Limit(50).Where("name like '%" + keys + "%'").Find(&pkgs)
	return pkgs, err
}

// UpdatePackage updates package's information.
func UpdatePackage(pkg *Package) error {
	pkg.Name = strings.ToLower(pkg.Name)
	_, err := x.Id(pkg.Id).AllCols().Update(pkg)
	return err
}
