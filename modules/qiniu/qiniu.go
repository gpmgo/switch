package qiniu

import (
	"strings"

	"github.com/qiniu/api.v6/io"
	"github.com/qiniu/api.v6/rs"

	"github.com/gpmgo/switch/modules/setting"
)

func GenUptoken() string {
	putPolicy := rs.PutPolicy{
		Scope: setting.BucketName,
	}
	return putPolicy.Token(nil)
}

// UploadArchive uploads local archive to QiNiu.
func UploadArchive(key, fpath string) error {
	uptoken := GenUptoken()
	if err := io.PutFile(nil, nil, uptoken, key, fpath, nil); err != nil {
		if !strings.Contains(err.Error(), `"code":614}`) {
			return err
		}
	}
	return nil
}

// DeleteArchive deletes a archive from QiNiu.
func DeleteArchive(key string) error {
	rsCli := rs.New(nil)
	return rsCli.Delete(nil, setting.BucketName, key)
}
