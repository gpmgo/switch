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
	"regexp"

	"github.com/gpmgo/switch/modules/setting"
)

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

// BlockRule represents a rule for blocking packages.
type BlockRule struct {
	Id   int64
	Rule string `xorm:"TEXT"`
	Note string
}

// NewBlockRule creates new block rule.
func NewBlockRule(r *BlockRule) error {
	_, err := x.Insert(r)
	return err
}

// ListBlockRules returns a list of block rules with given offset.
func ListBlockRules(offset int) ([]*BlockRule, error) {
	rules := make([]*BlockRule, 0, setting.PageSize)
	return rules, x.Limit(setting.PageSize, offset).Desc("id").Find(&rules)
}

// DeleteBlockRule deletes a block rule.
func DeleteBlockRule(id int64) error {
	_, err := x.Id(id).Delete(new(BlockRule))
	return err
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
