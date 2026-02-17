// this file was modified as of a derivative work of nxs-backup

package misc

import (
	"fmt"
	"math/rand"
	"net/http"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

type BackupType string

const (
	YearlyBackupDay  = "1"
	MonthlyBackupDay = "1"
	WeeklyBackupDay  = "0"

	DistributionFile = "ural-nxs-backup-" + runtime.GOARCH + ".tar.gz"
	LatestVersionURL = "https://github.com/uralm1/nxs-backup/releases/latest/download/" + DistributionFile
	VersionURL       = "https://github.com/uralm1/nxs-backup/releases/download/"

	DiscFiles            BackupType = "files"
	IncrFiles            BackupType = "incr_files"
	Mysql                BackupType = "mysql"
	MysqlXtrabackup      BackupType = "mysql_xtrabackup"
	MariadbBackup        BackupType = "mariadb_backup"
	Postgresql           BackupType = "postgresql"
	PostgresqlBasebackup BackupType = "postgresql_basebackup"
	MongoDB              BackupType = "mongodb"
	Redis                BackupType = "redis"
	External             BackupType = "external"
)

var DecadesBackupDays = []string{"1", "11", "21"}
var CPULimit = 0

func AllowedBackupTypesList() []string {
	return []string{
		string(DiscFiles),
		string(IncrFiles),
		string(Mysql),
		string(MysqlXtrabackup),
		string(MariadbBackup),
		string(Postgresql),
		string(PostgresqlBasebackup),
		string(MongoDB),
		string(Redis),
		string(External),
	}
}

func GetOfsPart(regex, target string) string {
	var pathParts []string

	regexParts := strings.Split(regex, "/")
	targetParts := strings.Split(target, "/")

	for i, p := range regexParts {
		if p != targetParts[i] {
			pathParts = append(pathParts, targetParts[i])
		}
	}

	if len(pathParts) > 0 {
		return strings.Join(pathParts, "___")
	}

	return targetParts[len(targetParts)-1]
}

// CurrentDateTimeFmt() returns "2026-01-22_16-41"
func CurrentDateTimeFmt() string {
	return time.Now().Format("2006-01-02_15-04")
}

// CurrentDayStr() returns current day as a string: "1-31(30)"
func CurrentDayStr() (Day string) {
	return strconv.Itoa(time.Now().Day())
}

// CurrentDOYStr() returns current day of the year as a string: "1-365(366)"
func CurrentDOYStr() (DOY string) {
	return strconv.Itoa(time.Now().YearDay())
}

// CurrentMonthStr() returns current month as a string: "1-12"
func CurrentMonthStr() (Month string) {
	return strconv.Itoa(int(time.Now().Month()))
}

// CurrentDOWStr() returns current day of the week as a string: "0-6"
func CurrentDOWStr() (DOW string) {
	return strconv.Itoa(int(time.Now().Weekday()))
}

// CurrentYearStr() returns current year as a string: "2026"
func CurrentYearStr() (Year string) {
	return strconv.Itoa(time.Now().Year())
}

// BeginningOfThisDay() returns time.Time of the beginning of local DAY
func BeginningOfThisDay() time.Time {
	t := time.Now()
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func GetDecadeDaySubdir() (decadeDay string) {
	intDom := time.Now().Day() //1-31(30)
	if intDom < 11 {
		decadeDay = "day_01"
	} else if intDom > 20 {
		decadeDay = "day_21"
	} else {
		decadeDay = "day_11"
	}
	return
}

func GetFileFullPath(dirPath, baseName, baseExtension, prefix string, gzip bool) (fullPath string) {
	fileName := fmt.Sprintf("%s_%s.%s", baseName, CurrentDateTimeFmt(), baseExtension)

	if prefix != "" {
		fileName = fmt.Sprintf("%s-%s", prefix, fileName)
	}

	if gzip {
		fileName += ".gz"
	}

	fullPath = filepath.Join(dirPath, fileName)

	return fullPath
}

// Contains() checks if a string is present in a slice
func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

// RandString generates random string
func RandString(strLen int64) string {
	var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, strLen)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}

	return string(b)
}

// CheckNewVersionAvailable checks if new version is available
func CheckNewVersionAvailable(major_ver string) ( /*newVer*/ string /*url*/, string, error) {
	var url string
	newVer, err := semver.NewVersion(major_ver)
	if err != nil {
		return "", "", err
	}
	curVer, err := semver.NewVersion(VERSION)
	if err != nil {
		return "", "", err
	}

	if major_ver != "13" {
		//https://github.com/uralm1/nxs-backup/releases/download/4/ural-nxs-backup-amd64.tar.gz
		url = VersionURL + newVer.String() + "/" + DistributionFile
	} else {
		//https://github.com/uralm1/nxs-backup/releases/latest/download/ural-nxs-backup-amd64.tar.gz
		//-> moved to https://github.com/uralm1/nxs-backup/releases/download/13.xx/nxs-backup
		url = LatestVersionURL
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("Failed to get new version from GitHub. Url: %s Status: %s ", url, resp.Status)
	}

	//referer is: https://github.com/uralm1/nxs-backup/releases/download/13.14/ural-nxs-backup-amd64.tar.gz
	re := regexp.MustCompile(`download/v?(\d+\.\d+(\.\d+(-[\w.]+)?)?)`)
	matches := re.FindStringSubmatch(resp.Request.Header.Get("Referer"))

	if len(matches) < 2 {
		return "", "", fmt.Errorf("no semver version found")
	}

	newVer, err = semver.NewVersion(matches[1])
	if err != nil {
		return "", "", fmt.Errorf("error while parsing version: %v", err)
	}
	//fmt.Printf("Latest version: %v\n", newVer)

	if curVer.LessThan(newVer) {
		return newVer.String(), url, nil
	}

	return "", "", nil
}
