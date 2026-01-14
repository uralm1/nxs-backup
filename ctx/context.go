// this file was modified as of a derivative work of nxs-backup

package ctx

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/uralm1/nxs-backup/modules/cmd_handler/list_backups"

	"github.com/docker/go-units"
	"github.com/sirupsen/logrus"

	"github.com/uralm1/nxs-backup/interfaces"
	"github.com/uralm1/nxs-backup/misc"
	"github.com/uralm1/nxs-backup/modules/cmd_handler/api_server"
	"github.com/uralm1/nxs-backup/modules/cmd_handler/generate_config"
	"github.com/uralm1/nxs-backup/modules/cmd_handler/start_backup"
	"github.com/uralm1/nxs-backup/modules/cmd_handler/test_config"
	"github.com/uralm1/nxs-backup/modules/logger"
	"github.com/uralm1/nxs-backup/modules/metrics"
)

// Ctx defines application custom context
type Ctx struct {
	Cmd       interfaces.Handler
	Log       *logrus.Logger
	Done      chan error
	EventCh   chan logger.LogRecord
	EventsWG  *sync.WaitGroup
	Notifiers []interfaces.Notifier
}

type app struct {
	waitTimeout time.Duration
	jobs        map[string]interfaces.Job
	fileJobs    interfaces.Jobs
	dbJobs      interfaces.Jobs
	extJobs     interfaces.Jobs
	initErr     error
	metricsData *metrics.Data
	serverBind  string
}

func AppCtxInit() (*Ctx, error) {
	c := &Ctx{
		EventsWG: &sync.WaitGroup{},
		EventCh:  make(chan logger.LogRecord),
		Done:     make(chan error),
	}

	ra, err := ReadArgs()
	if err != nil {
		return nil, err
	}

	c.Log = &logrus.Logger{
		Out:       os.Stderr,
		Level:     logrus.InfoLevel,
		Formatter: &logrus.TextFormatter{},
	}

	switch ra.Cmd {
	case update:
		fmt.Fprintln(os.Stderr, "Self update is not supported in this fork of nxs-backup.")
		return nil, errors.New("unsupported")
		//c.Cmd = self_update.Init(
		//	self_update.Opts{
		//		Version: ra.CmdParams.(*UpdateCmd).Version,
		//		Done:    c.Done,
		//	},
		//)

	case generate:
		if _, err = readConfig(ra.ConfigPath); err != nil {
			printInitError("Failed to read configuration file: %v\n", err)
			return nil, err
		}
		cp := ra.CmdParams.(*GenerateCmd)
		c.Cmd = generate_config.Init(
			generate_config.Opts{
				Done:     c.Done,
				CfgPath:  ra.ConfigPath,
				JobType:  cp.Type,
				OutPath:  cp.OutPath,
				Arg:      ra.Arg,
				Storages: cp.Storages,
			},
		)
	case testCfg:
		a, err := appInit(c, ra.ConfigPath)
		if err != nil {
			return nil, err
		}
		c.Cmd = test_config.Init(
			test_config.Opts{
				InitErr:  a.initErr,
				Done:     c.Done,
				FileJobs: a.fileJobs,
				DBJobs:   a.dbJobs,
				ExtJobs:  a.extJobs,
			},
		)
	case lsBackups:
		a, err := appInit(c, ra.ConfigPath)
		if err != nil {
			return nil, err
		}
		c.Cmd = list_backups.Init(
			list_backups.Opts{
				InitErr:  a.initErr,
				Done:     c.Done,
				JobName:  ra.CmdParams.(*StartCmd).JobName,
				FileJobs: a.fileJobs,
				DBJobs:   a.dbJobs,
				ExtJobs:  a.extJobs,
				Jobs:     a.jobs,
			},
		)
	case start:
		a, err := appInit(c, ra.ConfigPath)
		if err != nil {
			return nil, err
		}
		c.Cmd = start_backup.Init(
			start_backup.Opts{
				InitErr:     a.initErr,
				Done:        c.Done,
				EvCh:        c.EventCh,
				WaitPrev:    a.waitTimeout,
				JobName:     ra.CmdParams.(*StartCmd).JobName,
				Jobs:        a.jobs,
				FileJobs:    a.fileJobs,
				DBJobs:      a.dbJobs,
				ExtJobs:     a.extJobs,
				MetricsData: a.metricsData,
			},
		)
	case server:
		a, err := appInit(c, ra.ConfigPath)
		if err != nil {
			return nil, err
		}
		if a.metricsData == nil {
			err = fmt.Errorf("server metrics disabled by config")
			printInitError("Init err:\n%s", err)
			return nil, err
		}
		c.Cmd, err = api_server.Init(
			api_server.Opts{
				Bind:           a.serverBind,
				MetricFilePath: a.metricsData.MetricFilePath(),
				Log:            c.Log,
				Done:           c.Done,
			},
		)
		if err != nil {
			return nil, err
		}
	default:
		err = fmt.Errorf("unknown command: %s", ra.Cmd)
		printInitError("Init err:\n%s", err)
		return nil, err
	}

	return c, nil
}

func printInitError(ft string, err error) {
	_, _ = fmt.Fprintf(os.Stderr, ft, err)
}

func appInit(c *Ctx, cfgPath string) (app, error) {

	a := app{
		jobs: make(map[string]interfaces.Job),
	}

	conf, err := readConfig(cfgPath)
	if err != nil {
		printInitError("Failed to read configuration file: %v\n", err)
		return a, err
	}

	a.waitTimeout = conf.WaitingTimeout
	a.serverBind = conf.Server.Bind

	a.metricsData = metrics.InitData(
		metrics.DataOpts{
			Project:     conf.ProjectName,
			Server:      conf.ServerName,
			MetricsFile: conf.Server.Metrics.FilePath,
			Enabled:     false,
		},
	)

	if conf.Server.Metrics.Enabled {
		nva := 0.0
		// disabled in fork
		//ver, _ := semver.NewVersion(misc.VERSION)
		//newVer, _, _ := misc.CheckNewVersionAvailable(strconv.FormatUint(ver.Major(), 10))
		//if newVer != "" {
		//	nva = 1
		//}
		a.metricsData.NewVersionAvailable = nva
		a.metricsData.Enabled = true
	}

	if err = logInit(c, conf.LogFile, conf.LogLevel); err != nil {
		printInitError("Failed to init log file: %v\n", err)
		return a, err
	}

	// Notifications init
	if err = notifiersInit(c, conf); err != nil {
		a.initErr = errors.Join(a.initErr, err)
	}

	noLim := "0"
	lim := &limitsConf{
		NetRate:  &noLim,
		DiskRate: &noLim,
	}
	if conf.Limits != nil {
		if conf.Limits.NetRate != nil {
			lim.NetRate = conf.Limits.NetRate
		}
		if conf.Limits.DiskRate != nil {
			lim.DiskRate = conf.Limits.DiskRate
		}
		if conf.Limits.CPUCount != nil {
			misc.CPULimit = *conf.Limits.CPUCount
		}
	}

	// Init app
	storages, err := storagesInit(conf.StorageConnects, lim)
	if err != nil {
		a.initErr = errors.Join(a.initErr, err)
	}

	jobs, err := jobsInit(
		jobsOpts{
			jobs:        conf.Jobs,
			storages:    storages,
			metricsData: a.metricsData,
			mainLim:     lim,
		},
	)
	if err != nil {
		a.initErr = errors.Join(a.initErr, err)
	}

	for _, job := range jobs {
		switch job.GetType() {
		case misc.DiscFiles, misc.IncrFiles:
			a.fileJobs = append(a.fileJobs, job)
		case misc.Mysql, misc.MysqlXtrabackup, misc.MariadbBackup, misc.Postgresql, misc.PostgresqlBasebackup, misc.MongoDB, misc.Redis:
			a.dbJobs = append(a.dbJobs, job)
		case misc.External:
			a.extJobs = append(a.extJobs, job)
		}
		a.jobs[job.GetName()] = job
	}

	return a, nil
}

func logInit(c *Ctx, file, level string) error {
	var (
		f   *os.File
		l   logrus.Level
		err error
	)

	switch file {
	case "stdout":
		f = os.Stdout
	case "stderr":
		f = os.Stderr
	default:
		if err = os.MkdirAll(path.Dir(file), os.ModePerm); err != nil {
			return err
		}
		if f, err = os.OpenFile(file, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600); err != nil {
			return err
		}
	}

	// Validate log level
	if l, err = logrus.ParseLevel(level); err != nil {
		return fmt.Errorf("log init: %w", err)
	}

	//c.Log, err = appctx.DefaultLogInit(f, l, &logger.LogFormatter{})
	c.Log = &logrus.Logger{
		Out:       f,
		Level:     l,
		Formatter: &logger.LogFormatter{},
	}
	err = nil
	return err
}

func getRateLimit(limit *string) (rl int64, err error) {
	rl, err = units.FromHumanSize(*limit)
	if err != nil {
		return 0, fmt.Errorf("Failed to parse rate limit: %w. ", err)
	}

	return
}
