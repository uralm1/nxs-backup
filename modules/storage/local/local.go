package local

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/uralm1/nxs-backup/interfaces"
	"github.com/uralm1/nxs-backup/misc"
	"github.com/uralm1/nxs-backup/modules/backend/files"
	"github.com/uralm1/nxs-backup/modules/logger"
	. "github.com/uralm1/nxs-backup/modules/storage"
)

type Local struct {
	backupPath    string
	rateLimit     int64
	rotateEnabled bool
	Retention
}

func Init(rl int64) *Local {
	return &Local{
		rateLimit: rl,
	}
}

func (l *Local) Configure(p Params) {
	l.backupPath = p.BackupPath
	l.rateLimit = p.RateLimit
	l.rotateEnabled = p.RotateEnabled
	l.Retention = p.Retention
}

func (l *Local) IsLocal() int { return 1 }

func (l *Local) DeliverBackup(logCh chan logger.LogRecord, jobName, tmpBackupFile, ofs string, backupType misc.BackupType) (err error) {
	backupDstPath, metadataDstPath, links, err :=
		GetBackupDstAndLinks(tmpBackupFile, ofs, l.backupPath, l.Retention, backupType)
	if err != nil {
		logCh <- logger.Log(jobName, l.GetName()).Errorf("Unable to get destination path and links: '%s'", err)
		return
	}

	if metadataDstPath != "" { //this is actual only for incremental backup
		if err = l.deliverBackupMetadata(logCh, jobName, tmpBackupFile, metadataDstPath); err != nil {
			return
		}
	}

	err = os.MkdirAll(path.Dir(backupDstPath), os.ModePerm)
	if err != nil {
		logCh <- logger.Log(jobName, l.GetName()).Errorf("Unable to create directory: '%s'", err)
		return err
	}

	if err = os.Rename(tmpBackupFile, backupDstPath); err != nil {
		logCh <- logger.Log(jobName, l.GetName()).Debugf("Unable to move temp backup: %s", err)
		err = nil
		bakDst, err := os.Create(backupDstPath)
		if err != nil {
			return err
		}
		defer func() { _ = bakDst.Close() }()

		bakSrc, err := files.GetLimitedFileReader(tmpBackupFile, l.rateLimit)
		if err != nil {
			return err
		}
		defer func() { _ = bakSrc.Close() }()

		_, err = io.Copy(bakDst, bakSrc)
		if err != nil {
			logCh <- logger.Log(jobName, l.GetName()).Errorf("Unable to make copy: %s", err)
			return err
		}
		logCh <- logger.Log(jobName, l.GetName()).Infof("Successfully copied temp backup to %s", backupDstPath)
	} else {
		logCh <- logger.Log(jobName, l.GetName()).Infof("Successfully moved temp backup to %s", backupDstPath)
	}

	for dst, src := range links {
		err = os.MkdirAll(path.Dir(dst), os.ModePerm)
		if err != nil {
			logCh <- logger.Log(jobName, l.GetName()).Errorf("Unable to create directory: '%s'", err)
			return err
		}
		_ = os.Remove(dst)
		if err = os.Symlink(src, dst); err != nil {
			return err
		}
		logCh <- logger.Log(jobName, l.GetName()).Infof("Successfully created symlink %s", dst)
	}

	return
}

func (l *Local) deliverBackupMetadata(logCh chan logger.LogRecord, jobName, tmpBackupFile, metadataDstPath string) error {
	metadataSrcPath := tmpBackupFile + ".inc"

	err := os.MkdirAll(path.Dir(metadataDstPath), os.ModePerm)
	if err != nil {
		logCh <- logger.Log(jobName, l.GetName()).Errorf("Unable to create directory: '%s'", err)
		return err
	}

	_ = os.Remove(metadataDstPath)

	if err = os.Rename(metadataSrcPath, metadataDstPath); err != nil {
		logCh <- logger.Log(jobName, l.GetName()).Debugf("Unable to move temp backup: %s", err)

		metadataDst, err := os.Create(metadataDstPath)
		if err != nil {
			return err
		}
		defer func() { _ = metadataDst.Close() }()

		metadataSrc, err := files.GetLimitedFileReader(metadataSrcPath, l.rateLimit)
		if err != nil {
			return err
		}
		defer func() { _ = metadataSrc.Close() }()

		_, err = io.Copy(metadataDst, metadataSrc)
		if err != nil {
			logCh <- logger.Log(jobName, l.GetName()).Errorf("Unable to make copy: %s", err)
			return err
		}
		logCh <- logger.Log(jobName, l.GetName()).Infof("Successfully copied metadata to %s", metadataDstPath)
	} else {
		logCh <- logger.Log(jobName, l.GetName()).Infof("Successfully moved metadata to %s", metadataDstPath)
	}
	return nil
}

func (l *Local) DeleteOldBackups(logCh chan logger.LogRecord, ofsPart string, job interfaces.Job, full bool) error {
	if !l.rotateEnabled {
		logCh <- logger.Log(job.GetName(), l.GetName()).Info("Backup rotation was skipped (disabled in config)")
		return nil
	}

	if job.GetType() == misc.IncrFiles {
		return l.deleteIncrBackup(logCh, job.GetName(), ofsPart, full)
	} else {
		return l.deleteDiscBackup(logCh, job.GetName(), ofsPart, job.IsSafeRotation())
	}
}

func (l *Local) deleteDiscBackup(logCh chan logger.LogRecord, jobName, ofsPart string, safe_rotation bool) error {
	type fileLinks struct {
		wLink string
		dLink string
	}
	var errs []error
	filesMap := make(map[string]*fileLinks, 64)
	filesToDeleteMap := make(map[string]*fileLinks, 64)

	for _, p := range RetentionPeriodsList {
		retentionCount, retentionDate := GetRetention(p, l.Retention)
		if retentionCount == 0 && retentionDate.IsZero() {
			continue
		}

		bakDir := path.Join(l.backupPath, ofsPart, p.String())

		dir, err := os.Open(bakDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				logCh <- logger.Log(jobName, l.GetName()).Debugf("Backups directory `%s` not found. Continue.", bakDir)
				continue
			}
			logCh <- logger.Log(jobName, l.GetName()).Errorf("Failed to open directory '%s' with error: %s", bakDir, err)
			return err
		}

		lFiles, err := dir.ReadDir(-1)
		if err != nil {
			logCh <- logger.Log(jobName, l.GetName()).Errorf("Failed to read files in directory '%s' with error: %s", bakDir, err)
			return err
		}

		for _, file := range lFiles {
			fPath := path.Join(bakDir, file.Name())
			filesMap[fPath] = &fileLinks{}
			if file.Type()&fs.ModeSymlink != 0 {
				link, err := os.Readlink(fPath)
				if err != nil {
					logCh <- logger.Log(jobName, l.GetName()).Errorf("Failed to read a symlink for file '%s': %s",
						file, err)
					errs = append(errs, err)
					continue
				}
				linkPath := filepath.Join(bakDir, link)

				if fl, ok := filesMap[linkPath]; ok {
					switch p {
					case Weekly:
						fl.wLink = fPath
					case Daily:
						fl.dLink = fPath
					}
					filesMap[linkPath] = fl
				}
			}
		}

		if l.Retention.UseCount {
			sort.Slice(lFiles, func(i, j int) bool {
				iInfo, _ := lFiles[i].Info()
				jInfo, _ := lFiles[j].Info()
				return iInfo.ModTime().Before(jInfo.ModTime())
			})

			if !safe_rotation {
				retentionCount--
			}

			if retentionCount <= len(lFiles) {
				lFiles = lFiles[:len(lFiles)-retentionCount]
			} else {
				lFiles = lFiles[:0]
			}
		} else {
			i := 0
			for _, file := range lFiles {
				fileInfo, _ := file.Info()
				if fileInfo.ModTime().Location() != retentionDate.Location() {
					retentionDate = retentionDate.In(fileInfo.ModTime().Location())
				}

				if fileInfo.ModTime().Before(retentionDate) {
					lFiles[i] = file
					i++
				}
			}
			lFiles = lFiles[:i]
		}

		for _, file := range lFiles {
			if file.IsDir() {
				logCh <- logger.Log(jobName, l.GetName()).Warnf("`%s` is directory in %s. Please check and remove it.", file.Name(), bakDir)
				continue
			}
			fPath := path.Join(bakDir, file.Name())
			filesToDeleteMap[fPath] = filesMap[fPath]
		}
	}

	for file, fl := range filesToDeleteMap {
		delFile := true
		moved := false
		if fl.wLink != "" {
			if _, toDel := filesToDeleteMap[fl.wLink]; !toDel {
				delFile = false
				if err := moveFile(file, fl.wLink); err != nil {
					logCh <- logger.Log(jobName, l.GetName()).Error(err)
					errs = append(errs, err)
				} else {
					logCh <- logger.Log(jobName, l.GetName()).Debugf("Successfully moved old backup to %s", fl.wLink)
					moved = true
				}
				if _, toDel = filesToDeleteMap[fl.dLink]; !toDel {
					if err := os.Remove(fl.dLink); err != nil {
						logCh <- logger.Log(jobName, l.GetName()).Error(err)
						errs = append(errs, err)
						break
					}
					relative, _ := filepath.Rel(filepath.Dir(fl.dLink), fl.wLink)
					if err := os.Symlink(relative, fl.dLink); err != nil {
						logCh <- logger.Log(jobName, l.GetName()).Error(err)
						errs = append(errs, err)
					} else {
						logCh <- logger.Log(jobName, l.GetName()).Debugf("Successfully changed symlink %s", fl.dLink)
					}
				}
			}
		}
		if fl.dLink != "" && !moved {
			if _, toDel := filesToDeleteMap[fl.dLink]; !toDel {
				delFile = false
				if err := moveFile(file, fl.dLink); err != nil {
					logCh <- logger.Log(jobName, l.GetName()).Error(err)
					errs = append(errs, err)
				} else {
					logCh <- logger.Log(jobName, l.GetName()).Debugf("Successfully moved old backup to %s", fl.dLink)
				}
			}
		}

		if delFile {
			if err := os.Remove(file); err != nil {
				logCh <- logger.Log(jobName, l.GetName()).Errorf("Failed to delete file '%s' with error: %s",
					file, err)
				errs = append(errs, err)
			} else {
				logCh <- logger.Log(jobName, l.GetName()).Infof("Deleted old backup file '%s'", file)
			}
		}
	}

	return errors.Join(errs...)
}

func (l *Local) deleteIncrBackup(logCh chan logger.LogRecord, jobName, ofsPart string, full bool) error {
	var errs []error

	if full {
		backupDir := path.Join(l.backupPath, ofsPart)
		if err := os.RemoveAll(backupDir); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				logCh <- logger.Log(jobName, l.GetName()).Debugf("Directory '%s' not exist. Skipping delete.", backupDir)
				return nil
			} else {
				logCh <- logger.Log(jobName, l.GetName()).Errorf("Failed to delete '%s' with error: %s", backupDir, err)
				errs = append(errs, err)
			}
		}
	} else {
		intMoy, _ := strconv.Atoi(misc.GetDateTimeNow("moy"))
		lastMonth := intMoy - l.Months

		var year string
		if lastMonth > 0 {
			year = misc.GetDateTimeNow("year")
		} else {
			year = misc.GetDateTimeNow("previous_year")
			lastMonth += 12
		}

		backupDir := path.Join(l.backupPath, ofsPart, year)

		dirs, err := os.ReadDir(backupDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				logCh <- logger.Log(jobName, l.GetName()).Debugf("Directory '%s' not exist. Skipping rotate.", backupDir)
				return nil
			} else {
				logCh <- logger.Log(jobName, l.GetName()).Errorf("Failed to get access to directory '%s' with error: %v", backupDir, err)
				return err
			}
		}
		rx := regexp.MustCompile(`month_\d\d`)
		for _, dir := range dirs {
			dirName := dir.Name()
			if rx.MatchString(dirName) {
				dirParts := strings.Split(dirName, "_")
				dirMonth, _ := strconv.Atoi(dirParts[1])
				if dirMonth < lastMonth {
					if err = os.RemoveAll(path.Join(backupDir, dirName)); err != nil {
						logCh <- logger.Log(jobName, l.GetName()).Errorf("Failed to delete '%s' in dir '%s' with error: %s",
							dirName, backupDir, err)
						errs = append(errs, err)
					} else {
						logCh <- logger.Log(jobName, l.GetName()).Infof("Deleted old backup '%s' in directory '%s'", dirName, backupDir)
					}
				}
			}
		}
	}

	return errors.Join(errs...)
}

func (l *Local) GetFileReader(filePath string) (io.Reader, error) {
	fp, err := filepath.EvalSymlinks(path.Join(l.backupPath, filePath))
	if err != nil {
		return nil, err
	}
	return files.GetLimitedFileReader(fp, l.rateLimit)
}

func (l *Local) ListBackups(ofsPart string) ([]string, error) {
	backups := make([]string, 0)
	err := filepath.WalkDir(path.Join(l.backupPath, ofsPart), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		backups = append(backups, path)
		return nil
	})
	return backups, err
}

func (l *Local) Close() error {
	return nil
}

func (l *Local) Clone() interfaces.Storage {
	cl := *l
	return &cl
}

func (l *Local) GetName() string {
	return "local"
}

func moveFile(oldPath, newPath string) error {
	if err := os.Remove(newPath); err != nil {
		return fmt.Errorf("Failed to delete file '%s' with error: %s ", oldPath, err)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("Failed to move file '%s' with error: %s ", oldPath, err)
	}
	return nil
}
