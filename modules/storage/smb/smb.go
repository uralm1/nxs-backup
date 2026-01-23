package smb

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hirochachacha/go-smb2"

	"github.com/uralm1/nxs-backup/interfaces"
	"github.com/uralm1/nxs-backup/misc"
	"github.com/uralm1/nxs-backup/modules/backend/files"
	"github.com/uralm1/nxs-backup/modules/logger"
	. "github.com/uralm1/nxs-backup/modules/storage"
)

type SMB struct {
	session       *smb2.Session
	share         *smb2.Share
	name          string
	conn_params   Opts
	backupPath    string
	rateLimit     int64
	rotateEnabled bool
	Retention
}

type Opts struct {
	Host              string
	Port              int
	User              string
	Password          string
	Domain            string
	Share             string
	ConnectionTimeout time.Duration
}

func (s *SMB) connect_internal() error {
	conn, err := net.DialTimeout(
		"tcp",
		fmt.Sprintf("%s:%d", s.conn_params.Host, s.conn_params.Port),
		s.conn_params.ConnectionTimeout*time.Second,
	)
	if err != nil {
		return err
	}

	s.session, err = (&smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     s.conn_params.User,
			Password: s.conn_params.Password,
			Domain:   s.conn_params.Domain,
		},
	}).Dial(conn)
	if err != nil {
		return err
	}

	s.share, err = s.session.Mount(s.conn_params.Share)
	if err != nil {
		return fmt.Errorf("mount: %w", err)
	}
	return nil
}

func Init(sName string, params Opts, rl int64) (s *SMB, err error) {
	s = &SMB{
		name:        sName,
		conn_params: params,
		rateLimit:   rl,
	}

	if err := s.connect_internal(); err != nil {
		return s, fmt.Errorf("Failed to init '%s' SMB storage. Error: %v", sName, err)
	}
	return
}

func (s *SMB) Configure(p Params) {
	s.backupPath = strings.TrimPrefix(p.BackupPath, "/")
	s.rateLimit = p.RateLimit
	s.rotateEnabled = p.RotateEnabled
	s.Retention = p.Retention
}

func (s *SMB) IsLocal() int { return 0 }

func (s *SMB) DeliverBackup(logCh chan logger.LogRecord, jobName, tmpBackupFile, ofs string, backupType misc.BackupType) (err error) {
	backupDstPaths, metadataDstPaths :=
		GetBackupDstList(tmpBackupFile, ofs, s.backupPath, s.Retention, backupType)

	if len(metadataDstPaths) > 0 { //this is actual only for incremental backup
		for _, dstPath := range metadataDstPaths {
			if err = s.copy(logCh, jobName, tmpBackupFile+".inc", dstPath); err != nil {
				logCh <- logger.Log(jobName, s.name).Errorf("Unable to upload tmp backup (incremental)")
				return
			}
		}
	}

	for _, dstPath := range backupDstPaths {
		if err = s.copy(logCh, jobName, tmpBackupFile, dstPath); err != nil {
			logCh <- logger.Log(jobName, s.name).Errorf("Unable to upload tmp backup")
			return
		}
	}

	return nil
}

func (s *SMB) copy(logCh chan logger.LogRecord, jobName, srcPath, dstPath string) (err error) {
	// Make remote directories
	remDir := path.Dir(dstPath)

	//external backup could take a long time, so try to recreate
	// destination directory to determine a connection timeout
	// and implement a redial
	for attempts := 2; attempts > 0; attempts-- {
		err = s.share.MkdirAll(remDir, os.ModeDir)
		//reconnect here
		var connection_err *smb2.TransportError
		if errors.As(err, &connection_err) && attempts > 1 {
			s.Close()
			if err_retr := s.connect_internal(); err_retr != nil {
				logCh <- logger.Log(jobName, s.name).Errorf("Reconnection failed: '%s'", err_retr)
				//err = errors.Join(err, err_retr)
			} else {
				logCh <- logger.Log(jobName, s.name).Debugf("Reconnection succeeded")
				continue
			}
		}
		if err != nil {
			logCh <- logger.Log(jobName, s.name).Errorf("Unable to create remote directory '%s': '%s'", remDir, err)
			return
		} else {
			break
		}
	}

	dstFile, err := s.share.Create(dstPath)
	if err != nil {
		logCh <- logger.Log(jobName, s.name).Errorf("Unable to create remote file: %s", err)
		return
	}
	defer func() { _ = dstFile.Close() }()

	srcFile, err := files.GetLimitedFileReader(srcPath, s.rateLimit)
	if err != nil {
		logCh <- logger.Log(jobName, s.name).Errorf("Unable to open '%s'", err)
		return
	}
	defer func() { _ = srcFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		logCh <- logger.Log(jobName, s.name).Errorf("Unable to make copy: %s", err)
	} else {
		logCh <- logger.Log(jobName, s.name).Infof("File %s was successfully uploaded", dstPath)
	}
	return
}

func (s *SMB) DeleteOldBackups(logCh chan logger.LogRecord, ofsPart string, job interfaces.Job, full bool) error {
	if !s.rotateEnabled {
		logCh <- logger.Log(job.GetName(), s.name).Info("Backup rotation was skipped (disabled in config)")
		return nil
	}

	if job.GetType() == misc.IncrFiles {
		return s.deleteIncrBackup(logCh, job.GetName(), ofsPart, full)
	} else {
		return s.deleteDiscBackup(logCh, job.GetName(), ofsPart, job.IsSafeRotation())
	}
}

func (s *SMB) deleteDiscBackup(logCh chan logger.LogRecord, jobName, ofsPart string, safe_rotation bool) error {
	var errs []error

	for _, p := range RetentionPeriodsList {
		retentionCount, retentionDate := GetRetention(p, s.Retention)
		if retentionCount == 0 && retentionDate.IsZero() {
			continue
		}

		backupDir := path.Join(s.backupPath, ofsPart, p.String())
		smbFiles, err := s.share.ReadDir(backupDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			logCh <- logger.Log(jobName, s.name).Errorf("Failed to read files in remote directory '%s' with error: %s", backupDir, err)
			return err
		}

		if s.Retention.UseCount {
			sort.Slice(smbFiles, func(i, j int) bool {
				return smbFiles[i].ModTime().Before(smbFiles[j].ModTime())
			})

			if !safe_rotation {
				retentionCount--
			}
			if retentionCount <= len(smbFiles) {
				smbFiles = smbFiles[:len(smbFiles)-retentionCount]
			} else {
				smbFiles = smbFiles[:0]
			}
		} else {
			i := 0
			for _, file := range smbFiles {
				if file.ModTime().Location() != retentionDate.Location() {
					retentionDate = retentionDate.In(file.ModTime().Location())
				}

				if file.ModTime().Before(retentionDate) {
					smbFiles[i] = file
					i++
				}
			}
			smbFiles = smbFiles[:i]
		}

		for _, file := range smbFiles {
			if file.Name() == ".." || file.Name() == "." {
				continue
			}

			f_path := path.Join(backupDir, file.Name())
			if err := s.share.Remove(f_path); err != nil {
				logCh <- logger.Log(jobName, s.name).Errorf("Failed to delete file '%s' with error: %s", f_path, err)
				errs = append(errs, err)
			} else {
				logCh <- logger.Log(jobName, s.name).Infof("Deleted old backup file '%s'", f_path)
			}
		}
	}

	return errors.Join(errs...)
}

func (s *SMB) deleteIncrBackup(logCh chan logger.LogRecord, jobName, ofsPart string, full bool) error {
	var errs []error

	if full {
		backupDir := path.Join(s.backupPath, ofsPart)

		err := s.share.RemoveAll(backupDir)
		if err != nil {
			logCh <- logger.Log(jobName, s.name).Errorf("Failed to delete '%s' with error: %s", backupDir, err)
			errs = append(errs, err)
		}
	} else {
		intMoy, _ := strconv.Atoi(misc.GetDateTimeNow("moy"))
		lastMonth := intMoy - s.Months

		var year string
		if lastMonth > 0 {
			year = misc.GetDateTimeNow("year")
		} else {
			year = misc.GetDateTimeNow("previous_year")
			lastMonth += 12
		}

		backupDir := path.Join(s.backupPath, ofsPart, year)

		dirs, err := s.share.ReadDir(backupDir)
		if err != nil {
			logCh <- logger.Log(jobName, s.name).Errorf("Failed to get access to directory '%s' with error: %v", backupDir, err)
			return err
		}
		rx := regexp.MustCompile(`month_\d\d`)
		for _, dir := range dirs {
			dirName := dir.Name()
			if rx.MatchString(dirName) {
				dirParts := strings.Split(dirName, "_")
				dirMonth, _ := strconv.Atoi(dirParts[1])
				if dirMonth < lastMonth {
					if err = s.share.RemoveAll(path.Join(backupDir, dirName)); err != nil {
						logCh <- logger.Log(jobName, s.name).Errorf("Failed to delete '%s' in dir '%s' with error: %s",
							dirName, backupDir, err)
						errs = append(errs, err)
					} else {
						logCh <- logger.Log(jobName, s.name).Infof("Deleted old backup '%s' in directory '%s'", dirName, backupDir)
					}
				}
			}
		}
	}

	return errors.Join(errs...)
}

func (s *SMB) GetFileReader(ofsPath string) (io.Reader, error) {
	f, err := s.share.Open(path.Join(s.backupPath, ofsPath))
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

func (s *SMB) ListBackups(ofsPath string) ([]string, error) {
	bPath := path.Join(s.backupPath, ofsPath)

	fl, err := s.listFiles(bPath)
	if err != nil {
		return nil, err
	}

	return s.listPaths(bPath, fl)
}

func (s *SMB) listFiles(dstPath string) ([]fs.FileInfo, error) {

	smbEntries, err := s.share.ReadDir(dstPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = fmt.Errorf("%s: %v", dstPath, fs.ErrNotExist)
		}
		return nil, err
	}

	return smbEntries, nil
}

func (s *SMB) listPaths(base string, fList []fs.FileInfo) ([]string, error) {
	var paths []string

	for _, file := range fList {
		if !file.IsDir() {
			fp := path.Join(base, file.Name())
			paths = append(paths, fp)
		} else {
			subDir := path.Join(base, file.Name())
			subDirFiles, err := s.listFiles(subDir)
			if err != nil {
				return nil, err
			}
			subPaths, err := s.listPaths(subDir, subDirFiles)
			if err != nil {
				return nil, err
			}
			paths = append(paths, subPaths...)
		}
	}

	return paths, nil
}

func (s *SMB) Close() error {
	_ = s.share.Umount()
	return s.session.Logoff()
}

func (s *SMB) Clone() interfaces.Storage {
	cl := *s
	return &cl
}

func (s *SMB) GetName() string {
	return s.name
}
