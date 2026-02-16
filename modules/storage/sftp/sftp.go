package sftp

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

	"github.com/dustin/go-humanize"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/uralm1/nxs-backup/interfaces"
	"github.com/uralm1/nxs-backup/misc"
	"github.com/uralm1/nxs-backup/modules/backend/files"
	"github.com/uralm1/nxs-backup/modules/logger"
	. "github.com/uralm1/nxs-backup/modules/storage"
)

type SFTP struct {
	client        *sftp.Client
	name          string
	backupPath    string
	rateLimit     int64
	rotateEnabled bool
	Retention
}

type Opts struct {
	User           string
	Host           string
	Port           int
	Password       string
	KeyFile        string
	ConnectTimeout time.Duration
}

func Init(name string, opts Opts, rl int64) (*SFTP, error) {

	sshConfig := &ssh.ClientConfig{
		User:            opts.User,
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         opts.ConnectTimeout * time.Second,
		ClientVersion:   "SSH-2.0-" + "nxs-backup/" + misc.VERSION,
	}

	if opts.Password != "" {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(opts.Password))
	}

	// Load key file if specified
	if opts.KeyFile != "" {
		key, err := os.ReadFile(opts.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("Failed to init '%s' SFTP storage. Error: %v ", name, fmt.Errorf("failed to read private key file: %w", err))
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("Failed to init '%s' SFTP storage. Error: %v ", name, fmt.Errorf("failed to parse private key file: %w", err))
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(signer))
	}

	sshConn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", opts.Host, opts.Port), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to init '%s' SFTP storage. Error: %v ", name, fmt.Errorf("couldn't connect SSH: %w", err))
	}

	sftpClient, err := sftp.NewClient(sshConn)
	if err != nil {
		_ = sshConn.Close()
		return nil, fmt.Errorf("Failed to init '%s' SFTP storage. Error: %v ", name, fmt.Errorf("couldn't initialize SFTP: %w", err))
	}

	return &SFTP{
		name:      name,
		client:    sftpClient,
		rateLimit: rl,
	}, nil

}

func (s *SFTP) Configure(p Params) {
	s.backupPath = p.BackupPath
	s.rateLimit = p.RateLimit
	s.rotateEnabled = p.RotateEnabled
	s.Retention = p.Retention
}

func (s *SFTP) IsLocal() int { return 0 }

func (s *SFTP) DeliverBackup(logCh chan logger.LogRecord, jobName, tmpBackupFile, ofs string, backupType misc.BackupType) (err error) {
	backupDstPaths, metadataDstPaths :=
		GetBackupDstList(tmpBackupFile, ofs, s.backupPath, s.Retention, backupType)

	// len(metadataDstPaths) > 0 is actual only for incremental backup
	for _, dstPath := range metadataDstPaths {
		if err = s.copy(logCh, jobName, tmpBackupFile+".inc", dstPath); err != nil {
			logCh <- logger.Log(jobName, s.name).Errorf("Unable to upload incremental metadata file")
			return
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

func (s *SFTP) copy(logCh chan logger.LogRecord, job, src, dst string) (err error) {
	// Make remote directories
	dstDir := path.Dir(dst)

	if err = s.client.MkdirAll(dstDir); err != nil {
		logCh <- logger.Log(job, s.name).Errorf("Unable to create remote directory '%s': %s", dstDir, err)
		return
	}

	_ = s.client.Remove(dst)
	dstFile, err := s.client.Create(dst)
	if err != nil {
		logCh <- logger.Log(job, s.name).Errorf("Unable to create remote file: %s", err)
		return
	}
	defer func() { _ = dstFile.Close() }()

	srcFile, err := files.GetLimitedFileReader(src, s.rateLimit)
	if err != nil {
		logCh <- logger.Log(job, s.name).Errorf("Unable to open: %s", err)
		return
	}
	defer func() { _ = srcFile.Close() }()

	wr_bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		logCh <- logger.Log(job, s.name).Errorf("Unable to make copy: %s", err)
		return
	}

	logCh <- logger.Log(job, s.name).Infof("File %s was successfully uploaded (%s)", dst, humanize.Bytes(uint64(wr_bytes)))
	return nil
}

func (s *SFTP) DeleteOldBackups(logCh chan logger.LogRecord, ofsPart string, job interfaces.Job, full bool) error {
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

func (s *SFTP) deleteDiscBackup(logCh chan logger.LogRecord, jobName, ofsPart string, safe_rotation bool) error {
	var errs []error

	for _, p := range RetentionPeriodsList {
		retentionCount, retentionDate := GetRetention(p, s.Retention)
		if retentionCount == 0 && retentionDate.IsZero() {
			continue
		}

		backupDir := path.Join(s.backupPath, ofsPart, p.String())
		files, err := s.client.ReadDir(backupDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			logCh <- logger.Log(jobName, s.name).Errorf("Failed to read files in remote directory '%s' with error: %s", backupDir, err)
			return err
		}

		objs := NewRotationObjects(len(files))
		for _, file := range files {
			objs.AddObject(file.Name(), file.ModTime())
		}
		r_files := DGetRotatedObjects(objs, retentionCount, retentionDate, s.Retention.UseCount, safe_rotation)

		for _, file := range r_files {
			f_path := path.Join(backupDir, file)
			if err := s.client.Remove(f_path); err != nil {
				logCh <- logger.Log(jobName, s.name).Errorf("Failed to delete file '%s' with error: %s", f_path, err)
				errs = append(errs, err)
			} else {
				logCh <- logger.Log(jobName, s.name).Infof("Deleted old backup file '%s'", f_path)
			}
		}
	}

	return errors.Join(errs...)
}

func (s *SFTP) deleteIncrBackup(logCh chan logger.LogRecord, jobName, ofsPart string, full bool) error {
	var errs []error

	if full {
		backupDir := path.Join(s.backupPath, ofsPart)

		if err := s.client.Remove(backupDir); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				logCh <- logger.Log(jobName, s.name).Debugf("Directory '%s' is not exist. Deletion skipped.", backupDir)
				return nil
			}
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

		dirs, err := s.client.ReadDir(backupDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				logCh <- logger.Log(jobName, s.name).Debugf("Directory '%s' is not exist. Rotation skipped.", backupDir)
				return nil
			}
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
					if err = s.client.Remove(path.Join(backupDir, dirName)); err != nil {
						logCh <- logger.Log(jobName, s.name).Errorf("Failed to delete '%s' in directory '%s' with error: %s",
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

func (s *SFTP) GetFileReader(ofsPath string) (io.Reader, error) {
	f, err := s.client.Open(path.Join(s.backupPath, ofsPath))
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

func (s *SFTP) ListBackups(filePath string) (fl []string, err error) {
	walker := s.client.Walk(path.Join(s.backupPath, filePath))

	next := true
	for next {
		next = walker.Step()
		err = walker.Err()
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				err = fmt.Errorf("%s: %v", walker.Path(), err)
			}
			return
		}
		if !walker.Stat().IsDir() {
			fl = append(fl, walker.Path())
		}
	}

	return
}

func (s *SFTP) Close() error {
	return s.client.Close()
}

func (s *SFTP) Clone() interfaces.Storage {
	cl := *s
	return &cl
}

func (s *SFTP) GetName() string {
	return s.name
}
