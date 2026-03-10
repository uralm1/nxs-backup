package backup

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/uralm1/nxs-backup/interfaces"
	"github.com/uralm1/nxs-backup/misc"
	"github.com/uralm1/nxs-backup/modules/logger"
)

func Perform(logCh chan logger.LogRecord, job interfaces.Job) error {
	var errs []error
	var tmpDirPath string

	if !job.NeedToMakeBackup() {
		logCh <- logger.Log(job.GetName(), "").Infof("According to the backup plan today no new backups are created for the job %s", job.GetName())
		return nil
	}

	if job.GetStoragesCount() == 0 {
		logCh <- logger.Log(job.GetName(), "").Warn("There are no configured storages for the job.")
		return nil
	}

	if !job.IsSafeRotation() {
		if err := job.DeleteOldBackups(logCh, ""); err != nil {
			errs = append(errs, err)
		}
	}

	logCh <- logger.Log(job.GetName(), "").Infof("Starting %s job", job.GetName())

	if jobTmpDir := job.GetTempDir(); jobTmpDir != "" {
		tmpDirPath = path.Join(jobTmpDir, fmt.Sprintf("%s_%s", job.GetType(), misc.CurrentDateTimeFmt()))
		err := os.MkdirAll(tmpDirPath, os.ModePerm)
		if err != nil {
			logCh <- logger.Log(job.GetName(), "").Errorf("Job `%s` failed. Unable to create temporary directory, error: %s", job.GetName(), err)
			errs = append(errs, err)

			return errors.Join(errs...)
		}
	}

	if err := job.DoBackup(logCh, tmpDirPath); err != nil {
		errs = append(errs, err)
	}

	_ = job.CleanupTmpData()
	_ = filepath.Walk(tmpDirPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// try to delete empty dirs
			if info.IsDir() {
				_ = os.Remove(path)
			}
			return nil
		})
	// cleanup tmp dir
	_ = os.Remove(tmpDirPath)

	if job.IsSafeRotation() {
		if err := job.DeleteOldBackups(logCh, ""); err != nil {
			errs = append(errs, err)
		}
	}

	logCh <- logger.Log(job.GetName(), "").Info("Finished")

	return errors.Join(errs...)
}
