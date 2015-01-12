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
	"regexp"

	"github.com/gpmgo/switch/modules/archive"
	"github.com/gpmgo/switch/modules/log"
	"github.com/gpmgo/switch/modules/setting"
)

var (
	ErrBlockRuleNotExist = errors.New("Block rule does not exist")
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
func BlockPackage(importPath, note string) (keys []string, err error) {
	pkg, err := GetPakcageByPath(importPath)
	if err != nil {
		return nil, err
	}

	has, err := x.Where("import_path=?", pkg.ImportPath).Get(new(Block))
	if err != nil {
		return nil, err
	} else if has {
		return nil, nil
	}

	sess := x.NewSession()
	defer sess.Close()
	if err = sess.Begin(); err != nil {
		return nil, err
	}

	keys = make([]string, 0, 10)

	revs, err := pkg.GetRevisions()
	if err != nil {
		return nil, fmt.Errorf("error getting revisions(%s): %v", pkg.ImportPath, err)
	}

	ext := archive.GetExtension(pkg.ImportPath)
	for _, rev := range revs {
		switch rev.Storage {
		case QINIU:
			keys = append(keys, pkg.ImportPath+"-"+rev.Revision+ext)
		}
		if _, err = sess.Id(rev.Id).Delete(new(Revision)); err != nil {
			sess.Rollback()
			return nil, err
		}
	}
	os.RemoveAll(path.Join(setting.ArchivePath, pkg.ImportPath))

	if _, err = sess.Id(pkg.Id).Delete(new(Package)); err != nil {
		sess.Rollback()
		return nil, err
	}

	b := &Block{
		ImportPath: pkg.ImportPath,
		Note:       note,
	}
	if _, err = sess.Insert(b); err != nil {
		sess.Rollback()
		return nil, err
	}

	return keys, sess.Commit()
}

// ListBlockedPackages returns a list of block rules with given offset.
func ListBlockedPackages(offset int) ([]*Block, error) {
	blocks := make([]*Block, 0, setting.PageSize)
	return blocks, x.Limit(setting.PageSize, offset).Desc("id").Find(&blocks)
}

func UnblockPackage(id int64) error {
	_, err := x.Id(id).Delete(new(Block))
	return err
}

// BlockRule represents a rule for blocking packages.
type BlockRule struct {
	Id   int64
	Rule string `xorm:"UNIQUE"`
	Note string
}

// NewBlockRule creates new block rule.
func NewBlockRule(r *BlockRule) error {
	_, err := x.Insert(r)
	return err
}

// GetBlockRuleById returns a block rule by given ID.
func GetBlockRuleById(id int64) (*BlockRule, error) {
	r := new(BlockRule)
	has, err := x.Id(id).Get(r)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrBlockRuleNotExist
	}
	return r, nil
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

// RunBlockRule applies given block rule to all packages.
func RunBlockRule(id int64) (count int64, keys []string, err error) {
	r, err := GetBlockRuleById(id)
	if err != nil {
		return 0, nil, err
	}
	exp, err := regexp.Compile(r.Rule)
	if err != nil {
		return 0, nil, err
	}

	keys = make([]string, 0, 10)

	err = x.Iterate(new(Package), func(idx int, bean interface{}) error {
		pkg := bean.(*Package)

		if !exp.MatchString(pkg.ImportPath) {
			return nil
		}

		revs, err := pkg.GetRevisions()
		if err != nil {
			return fmt.Errorf("error getting revisions(%s): %v", pkg.ImportPath, err)
		}

		// Delete package archives.
		ext := archive.GetExtension(pkg.ImportPath)
		for _, rev := range revs {
			switch rev.Storage {
			case QINIU:
				keys = append(keys, pkg.ImportPath+"-"+rev.Revision+ext)
			}

			if _, err = x.Id(rev.Id).Delete(new(Revision)); err != nil {
				return fmt.Errorf("error deleting revision(%s-%s): %v", pkg.ImportPath, rev.Revision, err)
			}
		}
		os.RemoveAll(path.Join(setting.ArchivePath, pkg.ImportPath))

		if setting.ProdMode {
			if _, err = x.Id(pkg.Id).Delete(new(Package)); err != nil {
				return fmt.Errorf("error deleting package(%s): %v", pkg.ImportPath, err)
			}
		}

		log.Info("[%d] Package blocked: %s", r.Id, pkg.ImportPath)

		count++
		return nil
	})

	return count, keys, err
}
