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

package models

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Unknwon/com"

	"github.com/gpmgo/switch/modules/archive"
	"github.com/gpmgo/switch/modules/log"
	"github.com/gpmgo/switch/modules/qiniu"
	"github.com/gpmgo/switch/modules/setting"
)

var (
	ErrRevisionIsLocal  = errors.New("Revision archive is in local")
	ErrPackageNotExist  = errors.New("Package does not exist")
	ErrRevisionNotExist = errors.New("Revision does not exist")
)

type Storage int

const (
	LOCAL Storage = iota
	QINIU
)

// Revision represents a revision of a Go package.
type Revision struct {
	ID       int64    `xorm:"pk autoincr"`
	PkgID    int64    `xorm:"UNIQUE(s)"`
	Pkg      *Package `xorm:"-"`
	Revision string   `xorm:"UNIQUE(s)"`
	Storage
	Size    int64
	Updated time.Time `xorm:"UPDATED"`
}

func (r *Revision) GetPackage() (err error) {
	if r.Pkg != nil {
		return nil
	}
	r.Pkg, err = GetPakcageByID(r.PkgID)
	return err
}

// KeyName returns QiNiu key name.
func (r *Revision) KeyName() (string, error) {
	if r.Storage == LOCAL {
		return "", ErrRevisionIsLocal
	}
	if err := r.GetPackage(); err != nil {
		return "", err
	}
	return r.Pkg.ImportPath + "-" + r.Revision + archive.GetExtension(r.Pkg.ImportPath), nil
}

// GetRevision returns revision by given pakcage ID and revision.
func GetRevision(pkgID int64, rev string) (*Revision, error) {
	r := &Revision{
		PkgID:    pkgID,
		Revision: rev,
	}
	has, err := x.Get(r)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrRevisionNotExist
	}
	return r, nil
}

// UpdateRevision updates revision information.
func UpdateRevision(rev *Revision) error {
	_, err := x.Id(rev.ID).Update(rev)
	return err
}

// DeleteRevisionById delete revision by given ID.
func DeleteRevisionById(revId int64) error {
	_, err := x.Id(revId).Delete(new(Revision))
	return err
}

// GetLocalRevisions returns all revisions that archives are saved locally.
func GetLocalRevisions() ([]*Revision, error) {
	revs := make([]*Revision, 0, 10)
	err := x.Where("storage=0").Find(&revs)
	return revs, err
}

// GetRevisionsByPkgId returns a list of revisions of given package ID.
func GetRevisionsByPkgId(pkgId int64) ([]*Revision, error) {
	revs := make([]*Revision, 0, 10)
	err := x.Where("pkg_id=?", pkgId).Find(&revs)
	return revs, err
}

// Package represents a Go package.
type Package struct {
	ID             int64  `xorm:"pk autoincr"`
	ImportPath     string `xorm:"UNIQUE"`
	Description    string
	Homepage       string
	Issues         string
	DownloadCount  int64
	RecentDownload int64
	IsValidated    bool      `xorm:"DEFAULT 0"`
	Created        time.Time `xorm:"CREATED"`
}

func (pkg *Package) GetRevisions() ([]*Revision, error) {
	return GetRevisionsByPkgId(pkg.ID)
}

// NewPackage creates
func NewPackage(importPath string) (*Package, error) {
	pkg := &Package{
		ImportPath: importPath,
	}
	if _, err := x.Insert(pkg); err != nil {
		return nil, err
	}
	return pkg, nil
}

// GetPakcageByID returns a package by given ID.
func GetPakcageByID(pkgID int64) (*Package, error) {
	pkg := &Package{}
	has, err := x.Id(pkgID).Get(pkg)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrPackageNotExist
	}
	return pkg, nil
}

// GetPakcageByPath returns a package by given import path.
func GetPakcageByPath(importPath string) (*Package, error) {
	pkg := &Package{
		ImportPath: importPath,
	}
	has, err := x.Get(pkg)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrPackageNotExist
	}
	return pkg, nil
}

// CheckPkg checks if versioned package is in records, and download it when needed.
func CheckPkg(importPath, rev string) (*Revision, error) {
	// Check package record.
	pkg, err := GetPakcageByPath(importPath)
	if err != nil {
		if err != ErrPackageNotExist {
			return nil, err
		}
		blocked, blockErr, err := IsPackageBlocked(importPath)
		if err != nil {
			return nil, err
		} else if blocked {
			return nil, blockErr
		}
	}

	n := archive.NewNode(importPath, rev)

	// Get and check revision record.
	if err = n.GetRevision(); err != nil {
		return nil, err
	}

	var r *Revision
	if pkg != nil {
		r, err = GetRevision(pkg.ID, n.Revision)
		if err != nil && err != ErrRevisionNotExist {
			return nil, err
		}
	}

	return nil, fmt.Errorf("Revision: %s", n.Revision)

	if r == nil || (r.Storage == LOCAL && !com.IsFile(n.ArchivePath)) {
		if err := n.Download(); err != nil {
			return nil, err
		}
	}

	if pkg == nil {
		pkg, err = NewPackage(n.ImportPath)
		if err != nil {
			return nil, err
		}
	}

	if r == nil {
		r = &Revision{
			PkgID:    pkg.ID,
			Revision: n.Revision,
		}
		_, err = x.Insert(r)
	} else {
		_, err = x.Id(r.ID).Update(r)
	}
	return r, nil
}

// IncreasePackageDownloadCount increase package download count by 1.
func IncreasePackageDownloadCount(importPath string) error {
	pkg, err := GetPakcageByPath(importPath)
	if err != nil {
		return err
	}
	pkg.DownloadCount++
	pkg.RecentDownload++
	_, err = x.Id(pkg.ID).Update(pkg)
	return err
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

const _EXPIRE_DURATION = -1 * 24 * 30 * 3 * time.Hour

func cleanExpireRevesions() {
	if err := x.Where("updated<?", time.Now().Add(_EXPIRE_DURATION)).
		Iterate(new(Revision), func(idx int, bean interface{}) (err error) {
		rev := bean.(*Revision)
		if err = rev.GetPackage(); err != nil {
			return err
		}

		if _, err = x.Id(rev.ID).Delete(new(Revision)); err != nil {
			return err
		}

		ext := archive.GetExtension(rev.Pkg.ImportPath)
		fpath := path.Join(setting.ArchivePath, rev.Pkg.ImportPath, rev.Revision+ext)

		switch rev.Storage {
		case LOCAL:
			os.Remove(fpath)
			log.Info("Revision deleted: %s", fpath)
			return nil
		case QINIU:
			key, err := rev.KeyName()
			if err != nil {
				return err
			}
			if setting.ProdMode {
				if err = qiniu.DeleteArchive(key); err != nil {
					return err
				}
			}
			log.Info("Revision deleted: %s", key)
			return nil
		default:
			return nil
		}

		return nil
	}); err != nil {
		log.Error(3, "Fail to clean expire revisions: %v", err)
	}
}
