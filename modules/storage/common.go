package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
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
// returned:
//
//	retentionCount - same as retention setting of period asked,
//	retentionDate - date calculated back from current date (see code)
//
// example: "daily: 7", today is 02.06, retentionDate is 01.31
func GetRetention(p retentionPeriod, r Retention) (retentionCount int, retentionDate time.Time) {
	// set curDate to the beginning of local DAY
	t := time.Now()
	year, month, day := t.Date()
	curDate := time.Date(year, month, day, 0, 0, 0, 0, t.Location())

	switch p {
	case Daily:
		if r.Days == 0 {
			return
		}
		retentionCount = r.Days
		retentionDate = curDate.AddDate(0, 0, -r.Days+1)
	case Weekly:
		if misc.GetDateTimeNow("dow") != misc.WeeklyBackupDay || r.Weeks == 0 {
			return
		}
		retentionCount = r.Weeks
		retentionDate = curDate.AddDate(0, 0, -r.Weeks*7+1)
	case Monthly:
		if misc.GetDateTimeNow("dom") != misc.MonthlyBackupDay || r.Months == 0 {
			return
		}
		retentionCount = r.Months
		retentionDate = curDate.AddDate(0, -r.Months, 1)
	default:
		panic("Bad period")
	}
	return
}

func IsNeedToBackup(day, week, month int) bool {
	if day > 0 ||
		(week > 0 && misc.GetDateTimeNow("dow") == misc.WeeklyBackupDay) ||
		(month > 0 && misc.GetDateTimeNow("dom") == misc.MonthlyBackupDay) {
		return true
	}

	return false
}

// GetBackupDstAndLinks is a wrapper for getDBackupDstAndLinks and getIBackupDstAndLinks functions
func GetBackupDstAndLinks(tmpBackupFile, ofs, backupPath string, retention Retention, backupType misc.BackupType) (backupDst, metadataDst string, links map[string]string, err error) {
	if backupType == misc.IncrFiles {
		backupDst, metadataDst, links, err = getIBackupDstAndLinks(tmpBackupFile, ofs, backupPath)
	} else {
		backupDst, links, err = getDBackupDstAndLinks(tmpBackupFile, ofs, backupPath, retention)
		metadataDst = ""
	}
	return
}

// getDBackupDstAndLinks
// dst: "/backup/projpart/monthly/tmp.tar" (on 1st of month)
// links: "/backup/projpart/weekly/tmp.tar" -> "../monthly/tmp.tar" (on 1st of week)
// "/backup/projpart/daily/tmp.tar" -> "../monthly/tmp.tar"
// on other days it returns only dst daily path
func getDBackupDstAndLinks(tmpBackupFile, ofs, backupPath string, retention Retention) (dst string, links map[string]string, err error) {

	var relative string
	links = make(map[string]string)

	backupFileName := path.Base(tmpBackupFile)
	// first day of month
	if misc.GetDateTimeNow("dom") == misc.MonthlyBackupDay && retention.Months > 0 {
		dst = path.Join(backupPath, ofs, "monthly", backupFileName)
	}
	// first day of week (sunday)
	if misc.GetDateTimeNow("dow") == misc.WeeklyBackupDay && retention.Weeks > 0 {
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

func getIBackupDstAndLinks(tmpBackupFile, ofs, backupPath string) (backupDst, metadataDst string, links map[string]string, err error) {

	var relative string
	links = make(map[string]string)

	year := misc.GetDateTimeNow("year")
	dom := misc.GetDateTimeNow("dom")
	month := fmt.Sprintf("month_%02s", misc.GetDateTimeNow("moy"))
	decadeDay := misc.GetDecadeDaySubdir()

	init := true
	if _, err = os.Stat(tmpBackupFile + ".init"); errors.Is(err, fs.ErrNotExist) {
		init = false
		err = nil
	}

	backupFileName := path.Base(tmpBackupFile)
	// /backup/projpart/2026
	backupBasePath := path.Join(backupPath, ofs, year)
	// /backup/projpart/2026/inc_meta_info
	metadataPath := path.Join(backupBasePath, "inc_meta_info")

	if misc.GetDateTimeNow("doy") == misc.YearlyBackupDay || init {
		backupDst = path.Join(backupBasePath, "year", backupFileName)
		metadataDst = path.Join(metadataPath, "year.inc")
	}

	if dom == misc.MonthlyBackupDay || init {
		monthBackupDst := path.Join(backupBasePath, month, "monthly")
		if backupDst != "" {
			relative, err = filepath.Rel(monthBackupDst, backupDst)
			if err != nil {
				return
			}
			links[path.Join(monthBackupDst, backupFileName)] = relative
		} else {
			backupDst = path.Join(monthBackupDst, backupFileName)
		}
		monthMetadataDst := path.Join(metadataPath, "month.inc")
		if metadataDst != "" {
			relative, err = filepath.Rel(metadataPath, metadataDst)
			if err != nil {
				return
			}
			links[monthMetadataDst] = relative
		} else {
			metadataDst = monthMetadataDst
		}
	}

	dayDstPath := path.Join(backupBasePath, month, decadeDay)
	if backupDst != "" {
		relative, err = filepath.Rel(dayDstPath, backupDst)
		if err != nil {
			return
		}
		links[path.Join(dayDstPath, backupFileName)] = relative
	} else {
		backupDst = path.Join(dayDstPath, backupFileName)
	}
	if misc.Contains(misc.DecadesBackupDays, dom) || init {
		dayDst := path.Join(metadataPath, "day.inc")
		if metadataDst != "" {
			relative, err = filepath.Rel(metadataPath, metadataDst)
			if err != nil {
				return
			}
			links[dayDst] = relative
		} else {
			metadataDst = dayDst
		}
	}

	return
}

// GetBackupDstList is a wrapper for getDBackupDstList and getIBackupDstList functions
func GetBackupDstList(tmpBackupFile, ofs, backupPath string, retention Retention, backupType misc.BackupType) (backupDst, metadataDst []string) {
	if backupType == misc.IncrFiles {
		backupDst, metadataDst = getIBackupDstList(tmpBackupFile, ofs, backupPath)
	} else {
		backupDst = getDBackupDstList(tmpBackupFile, ofs, backupPath, retention)
		var e []string
		metadataDst = e
	}
	return
}

// getDBackupDstList
// dst: "/backup/projpart/monthly/tmp.tar", "/backup/projpart/weekly/tmp.tar", "/backup/projpart/daily/tmp.tar"
func getDBackupDstList(tmpBackupFile, ofs, backupPath string, retention Retention) (dst []string) {

	backupFile := path.Base(tmpBackupFile)
	basePath := path.Join(backupPath, ofs)
	// first day of month
	if misc.GetDateTimeNow("dom") == misc.MonthlyBackupDay && retention.Months > 0 {
		dst = append(dst, path.Join(basePath, "monthly", backupFile))
	}
	// first day of week (sunday)
	if misc.GetDateTimeNow("dow") == misc.WeeklyBackupDay && retention.Weeks > 0 {
		dst = append(dst, path.Join(basePath, "weekly", backupFile))
	}
	if retention.Days > 0 {
		dst = append(dst, path.Join(basePath, "daily", backupFile))
	}

	return
}

// getIBackupDstList
// backupDst: "/backup/projpart/2026/year/tmp.tar" (1st year), "/backup/projpart/2026/month_XX/monthly/tmp.tar" (1st month), "/backup/projpart/2026/month_XX/dayDD/tmp.tar"
// metadataDst: "/backup/projpart/2026/inc_meta_info/year.inc", "/backup/projpart/2026/inc_meta_info/month.inc", "/backup/projpart/2026/inc_meta_info/day.inc"
func getIBackupDstList(tmpBackupFile, ofs, backupPath string) (backupDst, metadataDst []string) {

	year := misc.GetDateTimeNow("year")
	dom := misc.GetDateTimeNow("dom")
	month := fmt.Sprintf("month_%02s", misc.GetDateTimeNow("moy"))
	decadeDay := misc.GetDecadeDaySubdir()

	init := true
	if _, err := os.Stat(tmpBackupFile + ".init"); errors.Is(err, fs.ErrNotExist) {
		init = false
	}

	backupFileName := path.Base(tmpBackupFile)
	// /backup/projpart/2026
	backupBasePath := path.Join(backupPath, ofs, year)
	// /backup/projpart/2026/inc_meta_info
	metadataPath := path.Join(backupBasePath, "inc_meta_info")

	if misc.GetDateTimeNow("doy") == misc.YearlyBackupDay || init {
		backupDst = append(backupDst, path.Join(backupBasePath, "year", backupFileName))
		metadataDst = append(metadataDst, path.Join(metadataPath, "year.inc"))
	}

	if dom == misc.MonthlyBackupDay || init {
		monthBackupDst := path.Join(backupBasePath, month, "monthly")
		backupDst = append(backupDst, path.Join(monthBackupDst, backupFileName))
		metadataDst = append(metadataDst, path.Join(metadataPath, "month.inc"))
	}

	dayDstPath := path.Join(backupBasePath, month, decadeDay)
	backupDst = append(backupDst, path.Join(dayDstPath, backupFileName))
	if misc.Contains(misc.DecadesBackupDays, dom) || init {
		metadataDst = append(metadataDst, path.Join(metadataPath, "day.inc"))
	}

	return
}

type RotateFileInfo struct {
	name    string
	modtime time.Time
}

type RotateFiles []RotateFileInfo

// DRotationFiles() takes list of files (RotateFiles structure) and returns list of file names that should be deleted
// retention_date, retention_count, safe_rotation are decision making parameters
func DRotationFiles(files RotateFiles, retention_date time.Time, retention_count int, use_count, safe_rotation bool) (names []string) {
	files = slices.DeleteFunc(files, func(f RotateFileInfo) bool {
		if f.name == ".." || f.name == "." || !(strings.HasSuffix(f.name, ".tar") || strings.HasSuffix(f.name, ".tar.gz")) {
			return true
		}
		return false
	})

	if use_count {
		if retention_count > 0 {
			sort.Slice(files, func(i, j int) bool {
				return files[i].modtime.Before(files[j].modtime)
			})

			if !safe_rotation {
				retention_count--
			}
			if retention_count <= len(files) {
				for _, f := range files[:len(files)-retention_count] {
					names = append(names, f.name)
				}
			} //else { names = []string{} }
		} else if retention_count == 0 {
			for _, f := range files {
				names = append(names, f.name)
			}
		}
	} else if !retention_date.IsZero() {
		for _, f := range files {
			if f.modtime.Location() != retention_date.Location() {
				retention_date = retention_date.In(f.modtime.Location())
			}

			if f.modtime.Before(retention_date) {
				names = append(names, f.name)
			}
		}
	}
	return //names
}
