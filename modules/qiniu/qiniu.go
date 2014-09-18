package qiniu

import (
	"os"
	"path"
	"strings"
	"time"

	"github.com/Unknwon/com"
	"github.com/qiniu/api/io"
	"github.com/qiniu/api/rs"

	"github.com/gpmgo/switch/models"
	"github.com/gpmgo/switch/modules/log"
	"github.com/gpmgo/switch/modules/setting"
)

func init() {
	if setting.ProdMode {
		go UploadArchives()
		ticker := time.NewTicker(time.Hour)
		go func() {
			for _ = range ticker.C {
				UploadArchives()
			}
		}()
	}
}

func genUptoken() string {
	putPolicy := rs.PutPolicy{
		Scope: setting.BucketName,
	}
	return putPolicy.Token(nil)
}

// UploadArchives checks and uploads local archives to QiNiu.
func UploadArchives() {
	revs, err := models.GetLocalRevisions()
	if err != nil {
		log.Error(5, "Fail to get local revisions: %v", err)
		return
	}

	// Upload.
	for _, rev := range revs {
		pkg, err := models.GetPakcageById(rev.PkgId)
		if err != nil {
			log.Error(5, "Fail to get package by ID(%d): %v", rev.PkgId, err)
			continue
		}

		uptoken := genUptoken()
		key := pkg.ImportPath + "-" + rev.Revision
		localPath := path.Join(pkg.ImportPath, rev.Revision)
		fpath := path.Join(setting.ArchivePath, localPath+".zip")

		if !com.IsFile(fpath) {
			log.Debug("Delete: %v", fpath)
			models.DeleteRevisionById(rev.Id)
			continue
		}

		// Check archive size.
		f, err := os.Open(fpath)
		if err != nil {
			log.Error(5, "Fail to open file(%s): %v", fpath, err)
			continue
		}
		fi, err := f.Stat()
		if err != nil {
			log.Error(5, "Fail to get file info(%s): %v", fpath, err)
			continue
		}
		// Greater then 5 MB.
		if fi.Size() > 5<<20 {
			log.Debug("Ignore large archive: %v", fpath)
			continue
		}

		log.Debug("Uploading: %s", localPath)
		if err := io.PutFile(nil, nil, uptoken, key, fpath, nil); err != nil {
			if !strings.Contains(err.Error(), `"code":614}`) {
				log.Error(5, "Fail to upload file(%s): %v", fpath, err)
				continue
			}
		}
		rev.Storage = models.QINIU
		if err := models.UpdateRevision(rev); err != nil {
			log.Error(5, "Fail to upadte revision(%d): %v", rev.Id, err)
			continue
		}
		os.Remove(fpath)
		log.Debug("Uploaded: %s", localPath)
	}
}
