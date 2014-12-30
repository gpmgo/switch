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
	"regexp"
	"strings"
	"time"

	"github.com/Unknwon/com"

	"github.com/gpmgo/switch/modules/archive"
)

var (
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
	Id       int64
	PkgId    int64  `xorm:"UNIQUE(s)"`
	Revision string `xorm:"UNIQUE(s)"`
	Storage
	Updated time.Time `xorm:"UPDATED"`
}

// GetRevision returns revision by given pakcage ID and revision.
func GetRevision(pkgId int64, rev string) (*Revision, error) {
	r := &Revision{
		PkgId:    pkgId,
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
	_, err := x.Id(rev.Id).Update(rev)
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
	Id             int64
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
	return GetRevisionsByPkgId(pkg.Id)
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

// GetPakcageById returns a package by given ID.
func GetPakcageById(pkgId int64) (*Package, error) {
	pkg := &Package{}
	has, err := x.Id(pkgId).Get(pkg)
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
		r, err = GetRevision(pkg.Id, n.Revision)
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
			PkgId:    pkg.Id,
			Revision: n.Revision,
		}
		_, err = x.Insert(r)
	} else {
		_, err = x.Id(r.Id).Update(r)
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
	_, err = x.Id(pkg.Id).Update(pkg)
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

// BlockError represents a block error which contains block note.
type BlockError struct {
	note string
}

func (e *BlockError) Error() string {
	return e.note
}

// Block represents information of a blocked package.
type Block struct {
	Id         int64
	ImportPath string `xorm:"UNIQUE"`
	Note       string
}

// BlockRule represents a rule for blocking packages.
type BlockRule struct {
	Id   int64
	Rule string `xorm:"TEXT"`
	Note string
}

// BlockPackage blocks given package.
func BlockPackage(pkg *Package, revs []*Revision, note string) (err error) {
	sess := x.NewSession()
	defer sess.Close()
	if err = sess.Begin(); err != nil {
		return err
	}

	for _, rev := range revs {
		if _, err = sess.Id(rev.Id).Delete(new(Revision)); err != nil {
			sess.Rollback()
			return err
		}
	}

	if _, err = sess.Id(pkg.Id).Delete(new(Package)); err != nil {
		sess.Rollback()
		return err
	}
	has, err := x.Where("import_path=?", pkg.ImportPath).Get(new(Block))
	if err != nil {
		return err
	} else if has {
		return nil
	}

	b := &Block{
		ImportPath: pkg.ImportPath,
		Note:       note,
	}
	if _, err = sess.Insert(b); err != nil {
		sess.Rollback()
		return err
	}

	return sess.Commit()
}

// IsPackageBlocked checks if a package is blocked.
func IsPackageBlocked(path string) (bool, error, error) {
	b := new(Block)
	has, err := x.Where("import_path=?", path).Get(b)
	if err != nil {
		return false, nil, err
	} else if has {
		return true, &BlockError{b.Note}, nil
	}

	if err = x.Iterate(new(BlockRule), func(idx int, bean interface{}) error {
		r := bean.(*BlockRule)
		exp, err := regexp.Compile(r.Rule)
		if err != nil {
			return err
		}
		if exp.MatchString(path) {
			return &BlockError{r.Note}
		}
		return nil
	}); err != nil {
		if _, ok := err.(*BlockError); ok {
			return true, err, nil
		}
		return false, nil, err
	}
	return false, nil, nil
}
