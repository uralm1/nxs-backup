package interfaces

import (
	"errors"
	"io"
	"os"
	"time"

	"github.com/uralm1/nxs-backup/modules/logger"
	"github.com/uralm1/nxs-backup/modules/metrics"
	"github.com/uralm1/nxs-backup/modules/storage"
)

type TargetFiles struct {
	List    []string
	ListErr error
}

type TargetsOnStorages map[string]TargetFiles

type Storage interface {
	Clone() Storage
	Configure(storage.Params)
	DeliverBackup(logCh chan logger.LogRecord, jobName, tmpBackupPath, ofs string) error
	DeleteOldBackups(logCh chan logger.LogRecord, ofsPart string, job Job) error
	GetFileReader(string) (io.Reader, error)
	GetName() string
	IsLocal() int
	ListBackups(string) ([]string, error)
	Close() error
}

type Storages []Storage

func (s Storages) Len() int           { return len(s) }
func (s Storages) Less(i, j int) bool { return s[i].IsLocal() < s[j].IsLocal() }
func (s Storages) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (s Storages) DeleteOldBackups(logCh chan logger.LogRecord, j Job, ofsPath string) error {
	var errs []error

	for _, st := range s {
		if ofsPath != "" {
			err := st.DeleteOldBackups(logCh, ofsPath, j)
			if err != nil {
				errs = append(errs, err)
			}
		} else {
			for _, ofsPart := range j.GetTargetOfsList() {
				err := st.DeleteOldBackups(logCh, ofsPart, j)
				if err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
	return errors.Join(errs...)
}

func (s Storages) Delivery(logCh chan logger.LogRecord, job Job) error {
	var errs []error

	for ofs, dumpObj := range job.GetDumpObjects() {
		if dumpObj.Delivered {
			continue
		}
		var deliveryErrs []error
		startTime := time.Now()
		ok := float64(0)
		for _, st := range s {
			if err := st.DeliverBackup(logCh, job.GetName(), dumpObj.TmpFile, ofs); err != nil {
				deliveryErrs = append(deliveryErrs, err)
			}
		}
		if len(deliveryErrs) == 0 {
			ok = float64(1)
		}
		job.SetOfsMetrics(ofs, map[string]float64{
			metrics.DeliveryOk:   ok,
			metrics.DeliveryTime: float64(time.Since(startTime).Nanoseconds() / 1e6),
		})
		if len(deliveryErrs) < len(s) {
			job.SetDumpObjectDelivered(ofs)
		}
		errs = append(errs, deliveryErrs...)
	}

	return errors.Join(errs...)
}

func (s Storages) ListBackups(ofs string) TargetsOnStorages {
	result := make(TargetsOnStorages)
	for _, st := range s {
		list, err := st.ListBackups(ofs)
		result[st.GetName()] = TargetFiles{
			List:    list,
			ListErr: err,
		}
	}

	return result
}

func (s Storages) CleanupTmpData(job Job) error {
	var errs []error

	for _, dumpObj := range job.GetDumpObjects() {
		tmpBackupFile := dumpObj.TmpFile
		// cleanup tmp backup file
		if err := os.Remove(tmpBackupFile); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s Storages) Close() error {
	for _, st := range s {
		_ = st.Close()
	}
	return nil
}
