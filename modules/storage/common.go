package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
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

func GetRetention(p retentionPeriod, r Retention) (retentionCount int, retentionDate time.Time) {
	curDate := time.Now().Round(24 * time.Hour)

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

func GetBackupDstAndLinks(tmpBackupFile, ofs, backupPath string, retention Retention, backupType misc.BackupType) (backupDst, metadataDst string, links map[string]string, err error) {
	if backupType == misc.IncrFiles {
		backupDst, metadataDst, links, err = getIBackupDstAndLinks(tmpBackupFile, ofs, backupPath)
	} else {
		backupDst, links, err = getDBackupDstAndLinks(tmpBackupFile, ofs, backupPath, retention)
		metadataDst = ""
	}
	return
}

func getDBackupDstAndLinks(tmpBackupFile, ofs, bakPath string, retention Retention) (dst string, links map[string]string, err error) {

	var relative string
	links = make(map[string]string)

	bakFileName := path.Base(tmpBackupFile)

	if misc.GetDateTimeNow("dom") == misc.MonthlyBackupDay && retention.Months > 0 {
		dst = path.Join(bakPath, ofs, "monthly", bakFileName)
	}
	if misc.GetDateTimeNow("dow") == misc.WeeklyBackupDay && retention.Weeks > 0 {
		dstPath := path.Join(bakPath, ofs, "weekly")
		if dst != "" {
			relative, err = filepath.Rel(dstPath, dst)
			if err != nil {
				return
			}
			links[path.Join(dstPath, bakFileName)] = relative
		} else {
			dst = path.Join(dstPath, bakFileName)
		}
	}
	if retention.Days > 0 {
		dstPath := path.Join(bakPath, ofs, "daily")
		if dst != "" {
			relative, err = filepath.Rel(dstPath, dst)
			if err != nil {
				return
			}
			links[path.Join(dstPath, bakFileName)] = relative
		} else {
			dst = path.Join(dstPath, bakFileName)
		}
	}

	return
}

func getIBackupDstAndLinks(tmpBackupFile, ofs, bakPath string) (bakDst, metadataDst string, links map[string]string, err error) {

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

	bakFileName := path.Base(tmpBackupFile)
	bakBasePath := path.Join(bakPath, ofs, year)
	metadataPath := path.Join(bakBasePath, "inc_meta_info")

	if misc.GetDateTimeNow("doy") == misc.YearlyBackupDay || init {
		bakDst = path.Join(bakBasePath, "year", bakFileName)
		metadataDst = path.Join(metadataPath, "year.inc")
	}

	if dom == misc.MonthlyBackupDay || init {
		monthBakDst := path.Join(bakBasePath, month, "monthly")
		if bakDst != "" {
			relative, err = filepath.Rel(monthBakDst, bakDst)
			if err != nil {
				return
			}
			links[path.Join(monthBakDst, bakFileName)] = relative
		} else {
			bakDst = path.Join(monthBakDst, bakFileName)
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

	dayDstPath := path.Join(bakBasePath, month, decadeDay)
	if bakDst != "" {
		relative, err = filepath.Rel(dayDstPath, bakDst)
		if err != nil {
			return
		}
		links[path.Join(dayDstPath, bakFileName)] = relative
	} else {
		bakDst = path.Join(dayDstPath, bakFileName)
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

func getDBackupDstList(tmpBackupFile, ofs, bakPath string, retention Retention) (dst []string) {

	bakFile := path.Base(tmpBackupFile)
	basePath := path.Join(bakPath, ofs)

	if misc.GetDateTimeNow("dom") == misc.MonthlyBackupDay && retention.Months > 0 {
		dst = append(dst, path.Join(basePath, "monthly", bakFile))
	}
	if misc.GetDateTimeNow("dow") == misc.WeeklyBackupDay && retention.Weeks > 0 {
		dst = append(dst, path.Join(basePath, "weekly", bakFile))
	}
	if retention.Days > 0 {
		dst = append(dst, path.Join(basePath, "daily", bakFile))
	}

	return
}

func getIBackupDstList(tmpBackupFile, ofs, bakPath string) (bakDst, metadataDst []string) {

	year := misc.GetDateTimeNow("year")
	dom := misc.GetDateTimeNow("dom")
	month := fmt.Sprintf("month_%02s", misc.GetDateTimeNow("moy"))
	decadeDay := misc.GetDecadeDaySubdir()

	init := true
	if _, err := os.Stat(tmpBackupFile + ".init"); errors.Is(err, fs.ErrNotExist) {
		init = false
	}

	bakFileName := path.Base(tmpBackupFile)
	bakBasePath := path.Join(bakPath, ofs, year)
	metadataPath := path.Join(bakBasePath, "inc_meta_info")

	if misc.GetDateTimeNow("doy") == misc.YearlyBackupDay || init {
		bakDst = append(bakDst, path.Join(bakBasePath, "year", bakFileName))
		metadataDst = append(metadataDst, path.Join(metadataPath, "year.inc"))
	}

	if dom == misc.MonthlyBackupDay || init {
		monthBakDst := path.Join(bakBasePath, month, "monthly")
		bakDst = append(bakDst, path.Join(monthBakDst, bakFileName))
		metadataDst = append(metadataDst, path.Join(metadataPath, "month.inc"))
	}

	dayDstPath := path.Join(bakBasePath, month, decadeDay)
	bakDst = append(bakDst, path.Join(dayDstPath, bakFileName))
	if misc.Contains(misc.DecadesBackupDays, dom) || init {
		metadataDst = append(metadataDst, path.Join(metadataPath, "day.inc"))
	}

	return
}
