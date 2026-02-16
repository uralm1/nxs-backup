package webdav

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/uralm1/nxs-backup/interfaces"
	"github.com/uralm1/nxs-backup/misc"
	"github.com/uralm1/nxs-backup/modules/backend/files"
	"github.com/uralm1/nxs-backup/modules/backend/webdav"
	"github.com/uralm1/nxs-backup/modules/logger"
	. "github.com/uralm1/nxs-backup/modules/storage"
)

type WebDav struct {
	client        *webdav.Client
	name          string
	backupPath    string
	rateLimit     int64
	rotateEnabled bool
	Retention
}

type Opts struct {
	URL               string
	Username          string
	Password          string
	OAuthToken        string
	ConnectionTimeout time.Duration
}

func Init(name string, params Opts, rl int64) (*WebDav, error) {

	client, err := webdav.Init(webdav.Params{
		URL:               params.URL,
		Username:          params.Username,
		Password:          params.Password,
		OAuthToken:        params.OAuthToken,
		ConnectionTimeout: params.ConnectionTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to init '%s' WebDav storage. Error: %v ", name, err)
	}

	return &WebDav{
		name:      name,
		client:    client,
		rateLimit: rl,
	}, nil
}

func (wd *WebDav) Configure(p Params) {
	wd.backupPath = path.Join("/", p.BackupPath)
	wd.rateLimit = p.RateLimit
	wd.rotateEnabled = p.RotateEnabled
	wd.Retention = p.Retention
}

func (wd *WebDav) IsLocal() int { return 0 }

func (wd *WebDav) DeliverBackup(logCh chan logger.LogRecord, jobName, tmpBackupFile, ofs string, backupType misc.BackupType) (err error) {
	backupDstPaths, metadataDstPaths :=
		GetBackupDstList(tmpBackupFile, ofs, wd.backupPath, wd.Retention, backupType)

	// len(metadataDstPaths) > 0 is actual only for incremental backup
	for _, dstPath := range metadataDstPaths {
		if err = wd.copy(logCh, jobName, tmpBackupFile+".inc", dstPath); err != nil {
			logCh <- logger.Log(jobName, wd.name).Errorf("Unable to upload incremental metadata file")
			return
		}
	}

	for _, dstPath := range backupDstPaths {
		if err = wd.copy(logCh, jobName, tmpBackupFile, dstPath); err != nil {
			logCh <- logger.Log(jobName, wd.name).Errorf("Unable to upload tmp backup")
			return
		}
	}

	return nil
}

func (wd *WebDav) copy(logCh chan logger.LogRecord, job, src, dst string) (err error) {
	// Make remote directories
	remDir := path.Dir(dst)
	if err = wd.mkDir(remDir); err != nil {
		logCh <- logger.Log(job, wd.name).Errorf("Unable to create remote directory '%s': %s", remDir, err)
		return
	}

	srcFile, err := files.GetLimitedFileReader(src, wd.rateLimit)
	if err != nil {
		logCh <- logger.Log(job, wd.name).Errorf("Unable to open: %s", err)
		return
	}
	defer func() { _ = srcFile.Close() }()

	err = wd.client.Upload(dst, srcFile)
	if err != nil {
		logCh <- logger.Log(job, wd.name).Errorf("Unable to upload file: %s", err)
	} else {
		logCh <- logger.Log(job, wd.name).Infof("File %s was successfull uploaded", dst)
	}

	return
}

func (wd *WebDav) DeleteOldBackups(logCh chan logger.LogRecord, ofsPart string, job interfaces.Job, full bool) error {
	if !wd.rotateEnabled {
		logCh <- logger.Log(job.GetName(), wd.name).Info("Backup rotation was skipped (disabled in config)")
		return nil
	}

	if job.GetType() == misc.IncrFiles {
		return wd.deleteIncrBackup(logCh, job.GetName(), ofsPart, full)
	} else {
		return wd.deleteDiscBackup(logCh, job.GetName(), ofsPart, job.IsSafeRotation())
	}
}

func (wd *WebDav) deleteDiscBackup(logCh chan logger.LogRecord, jobName, ofsPart string, safe_rotation bool) error {
	var errs []error

	for _, p := range RetentionPeriodsList {
		retentionCount, retentionDate := GetRetention(p, wd.Retention)
		if retentionCount == 0 && retentionDate.IsZero() {
			continue
		}

		backupDir := path.Join(wd.backupPath, ofsPart, p.String())
		wdFiles, err := wd.client.Ls(backupDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			logCh <- logger.Log(jobName, wd.name).Errorf("Failed to read files in remote directory '%s' with error: %s", backupDir, err)
			return err
		}

		objs := NewRotationObjects(len(wdFiles))
		for _, file := range wdFiles {
			objs.AddObject(file.Name(), file.ModTime())
		}
		r_files := DGetRotatedObjects(objs, retentionCount, retentionDate, wd.Retention.UseCount, safe_rotation)

		for _, file := range r_files {
			err = wd.client.Rm(path.Join(backupDir, file))
			if err != nil {
				logCh <- logger.Log(jobName, wd.name).Errorf("Failed to delete file '%s' in directory '%s' with error: %s",
					file, backupDir, err)
				errs = append(errs, err)
			} else {
				logCh <- logger.Log(jobName, wd.name).Infof("Deleted old backup file '%s' in directory '%s'", file, backupDir)
			}
		}
	}

	return errors.Join(errs...)
}

func (wd *WebDav) deleteIncrBackup(logCh chan logger.LogRecord, jobName, ofsPart string, full bool) error {
	var errs []error

	if full {
		backupDir := path.Join(wd.backupPath, ofsPart)

		err := wd.client.Rm(backupDir)
		if err != nil {
			logCh <- logger.Log(jobName, wd.name).Errorf("Failed to delete '%s' with error: %s", backupDir, err)
			errs = append(errs, err)
		}
	} else {
		intMoy, _ := strconv.Atoi(misc.GetDateTimeNow("moy"))
		lastMonth := intMoy - wd.Months

		var year string
		if lastMonth > 0 {
			year = misc.GetDateTimeNow("year")
		} else {
			year = misc.GetDateTimeNow("previous_year")
			lastMonth += 12
		}

		backupDir := path.Join(wd.backupPath, ofsPart, year)

		dirs, err := wd.client.Ls(backupDir)
		if err != nil {
			logCh <- logger.Log(jobName, wd.name).Errorf("Failed to get access to directory '%s' with error: %v", backupDir, err)
			return err
		}
		rx := regexp.MustCompile(`month_\d\d`)
		for _, dir := range dirs {
			dirName := dir.Name()
			if rx.MatchString(dirName) {
				dirParts := strings.Split(dirName, "_")
				dirMonth, _ := strconv.Atoi(dirParts[1])
				if dirMonth < lastMonth {
					if err = wd.client.Rm(path.Join(backupDir, dirName)); err != nil {
						logCh <- logger.Log(jobName, wd.name).Errorf("Failed to delete '%s' in dir '%s' with error: %s",
							dirName, backupDir, err)
						errs = append(errs, err)
					} else {
						logCh <- logger.Log(jobName, wd.name).Infof("Deleted old backup '%s' in directory '%s'", dirName, backupDir)
					}
				}
			}
		}
	}

	return errors.Join(errs...)
}

func (wd *WebDav) mkDir(dstPath string) error {

	dstPath = path.Clean(dstPath)
	if dstPath == "." || dstPath == "/" {
		return nil
	}
	fi, err := wd.getInfo(dstPath)
	if err == nil {
		if fi.IsDir() {
			return nil
		}
		return fmt.Errorf("%s is a file not a directory", dstPath)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("mkdir %q failed: %w", dstPath, err)
	}

	dir := path.Dir(dstPath)
	err = wd.mkDir(dir)
	if err != nil {
		return err
	}
	err = wd.client.Mkdir(dstPath)
	if err != nil {
		return err
	}

	return nil
}

func (wd *WebDav) getInfo(dstPath string) (os.FileInfo, error) {

	dir := path.Dir(dstPath)
	base := path.Base(dstPath)

	wdfl, err := wd.client.Ls(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range wdfl {
		if file.Name() == base {
			return file, nil
		}
	}
	return nil, fs.ErrNotExist
}

func (wd *WebDav) GetFileReader(ofsPath string) (io.Reader, error) {
	f, err := wd.client.Read(path.Join(wd.backupPath, ofsPath))
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var buf []byte
	buf, err = io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buf), err
}

func (wd *WebDav) ListBackups(ofsPath string) ([]string, error) {
	bPath := path.Join(wd.backupPath, ofsPath)

	fl, err := wd.client.Ls(bPath)
	if err != nil {
		return nil, err
	}

	return wd.listPaths(bPath, fl)
}

func (wd *WebDav) listPaths(base string, fList []fs.FileInfo) ([]string, error) {
	var paths []string

	for _, file := range fList {
		if !file.IsDir() {
			fp := path.Join(base, file.Name())
			paths = append(paths, fp)
		} else {
			subDir := path.Join(base, file.Name())
			subDirFiles, err := wd.client.Ls(subDir)
			if err != nil {
				return nil, err
			}
			subPaths, err := wd.listPaths(subDir, subDirFiles)
			if err != nil {
				return nil, err
			}
			paths = append(paths, subPaths...)
		}
	}

	return paths, nil
}

func (wd *WebDav) Close() error {
	return nil
}

func (wd *WebDav) Clone() interfaces.Storage {
	cl := *wd
	return &cl
}

func (wd *WebDav) GetName() string {
	return wd.name
}
