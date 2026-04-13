package storage

import (
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/uralm1/nxs-backup/misc"
)

type retentionPeriod string

const (
	Daily   retentionPeriod = "daily"
	Weekly  retentionPeriod = "weekly"
	Monthly retentionPeriod = "monthly"
)

var RetentionPeriodsList = []retentionPeriod{Monthly, Weekly, Daily}

type Params struct {
	RateLimit     int64
	BackupPath    string
	RotateEnabled bool
	Retention
}

type Retention struct {
	Days     int
	Weeks    int
	Months   int
	UseCount bool
}

func (p retentionPeriod) String() string {
	return string(p)
}

// GetRetention() retrives information from the retention setting (r) of a storage for a period "daily","weekly","monthly" (p)
// returns:
// retentionCount - same as retention setting of period asked,
// retentionDate - date calculated back from current date (see code)
//
// example: "daily: 7", today is 02.06, retentionDate is 01.31
func GetRetention(p retentionPeriod, r Retention) (retentionCount int, retentionDate time.Time) {
	curDate := misc.BeginningOfThisDay()

	switch p {
	case Daily:
		if r.Days == 0 {
			return
		}
		retentionCount = r.Days
		retentionDate = curDate.AddDate(0, 0, -r.Days+1)
	case Weekly:
		if misc.CurrentDOWStr() != misc.WeeklyBackupDay || r.Weeks == 0 {
			return
		}
		retentionCount = r.Weeks
		retentionDate = curDate.AddDate(0, 0, -r.Weeks*7+1)
	case Monthly:
		if misc.CurrentDayStr() != misc.MonthlyBackupDay || r.Months == 0 {
			return
		}
		retentionCount = r.Months
		retentionDate = curDate.AddDate(0, -r.Months, 1)
	default:
		panic("Bad period")
	}
	return
}

func IsNeedToBackup(r Retention) bool {
	if r.Days > 0 ||
		(r.Weeks > 0 && misc.CurrentDOWStr() == misc.WeeklyBackupDay) ||
		(r.Months > 0 && misc.CurrentDayStr() == misc.MonthlyBackupDay) {
		return true
	}

	return false
}

// GetDBackupDstAndLinks
// dst: "/backup/projpart/monthly/tmp.tar" (on 1st of month)
// links: "/backup/projpart/weekly/tmp.tar" -> "../monthly/tmp.tar" (on 1st of week)
// "/backup/projpart/daily/tmp.tar" -> "../monthly/tmp.tar"
// on other days it returns only dst daily path
func GetBackupDstAndLinks(tmpBackupFile, ofs, backupPath string, retention Retention) (dst string, links map[string]string, err error) {

	var relative string
	links = make(map[string]string)

	backupFileName := path.Base(tmpBackupFile)
	// first day of month
	if misc.CurrentDayStr() == misc.MonthlyBackupDay && retention.Months > 0 {
		dst = path.Join(backupPath, ofs, "monthly", backupFileName)
	}
	// first day of week (sunday)
	if misc.CurrentDOWStr() == misc.WeeklyBackupDay && retention.Weeks > 0 {
		dstPath := path.Join(backupPath, ofs, "weekly")
		if dst != "" {
			relative, err = filepath.Rel(dstPath, dst)
			if err != nil {
				return
			}
			links[path.Join(dstPath, backupFileName)] = relative
		} else {
			dst = path.Join(dstPath, backupFileName)
		}
	}
	if retention.Days > 0 {
		dstPath := path.Join(backupPath, ofs, "daily")
		if dst != "" {
			relative, err = filepath.Rel(dstPath, dst)
			if err != nil {
				return
			}
			links[path.Join(dstPath, backupFileName)] = relative
		} else {
			dst = path.Join(dstPath, backupFileName)
		}
	}

	return
}

// GetBackupDstList
// dst: "/backup/projpart/monthly/tmp.tar", "/backup/projpart/weekly/tmp.tar", "/backup/projpart/daily/tmp.tar"
func GetBackupDstList(tmpBackupFile, ofs, backupPath string, retention Retention) (dst []string) {

	backupFile := path.Base(tmpBackupFile)
	basePath := path.Join(backupPath, ofs)
	// first day of month
	if misc.CurrentDayStr() == misc.MonthlyBackupDay && retention.Months > 0 {
		dst = append(dst, path.Join(basePath, "monthly", backupFile))
	}
	// first day of week (sunday)
	if misc.CurrentDOWStr() == misc.WeeklyBackupDay && retention.Weeks > 0 {
		dst = append(dst, path.Join(basePath, "weekly", backupFile))
	}
	if retention.Days > 0 {
		dst = append(dst, path.Join(basePath, "daily", backupFile))
	}

	return
}

type RotationObjectInfo struct {
	name    string
	modtime time.Time
}

type RotationObjects []RotationObjectInfo

func NewRotationObjects(cap int) RotationObjects {
	return make([]RotationObjectInfo, 0, cap)
}

func (objs *RotationObjects) AddObject(name string, modtime time.Time) {
	*objs = append(*objs, RotationObjectInfo{name, modtime})
}

// DGetRotatedObjects() takes list of objects (files, RotationObjects structure) and returns list of object names that should be deleted
// retention_count, retention_date, use_count, safe_rotation are decision making parameters
func DGetRotatedObjects(objects RotationObjects, retention_count int, retention_date time.Time, use_count, safe_rotation bool) []string {
	objects = slices.DeleteFunc(objects, func(o RotationObjectInfo) bool {
		if o.name == ".." || o.name == "." || !(strings.HasSuffix(o.name, ".tar") || strings.HasSuffix(o.name, ".tar.gz")) {
			return true
		}
		return false
	})

	names := make([]string, 0, len(objects))

	if use_count {
		if retention_count > 0 {
			sort.Slice(objects, func(i, j int) bool {
				return objects[i].modtime.Before(objects[j].modtime)
			})

			if !safe_rotation {
				retention_count--
			}
			if retention_count <= len(objects) {
				for _, o := range objects[:len(objects)-retention_count] {
					names = append(names, o.name)
				}
			} //else { names = []string{} }
		} else if retention_count == 0 {
			for _, o := range objects {
				names = append(names, o.name)
			}
		}
	} else if !retention_date.IsZero() {
		for _, o := range objects {
			if o.modtime.Location() != retention_date.Location() {
				retention_date = retention_date.In(o.modtime.Location())
			}

			if o.modtime.Before(retention_date) {
				names = append(names, o.name)
			}
		}
	}
	return names
}
