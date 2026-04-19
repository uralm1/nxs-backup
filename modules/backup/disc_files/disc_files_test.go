package disc_files

import (
	"fmt"
	"testing"

	"github.com/uralm1/nxs-backup/modules/metrics"
)

func TestInit(t *testing.T) {
	var sources []SourceParams
	sources = append(sources, SourceParams{
		Name:        "test_source",
		Targets:     []string{"/home/test/aaa", "/home/test/bbb"},
		Excludes:    []string{},
		Gzip:        true,
		SaveAbsPath: false,
	})

	metrics_data_ptr := metrics.InitData(
		metrics.DataOpts{
			Project:             "ProjectName",
			Server:              "ServerName",
			MetricsFile:         "Server.Metrics.FilePath",
			Enabled:             false,
			NewVersionAvailable: 0,
		},
	)

	jp := JobParams{
		Name:             "test",
		TmpDir:           "/home/test/tmp",
		NeedToMakeBackup: true,
		SafeRotation:     false,
		DeferredCopying:  false,
		DiskRateLimit:    0,
		//Storages:       [],
		Sources: sources,
		Metrics: metrics_data_ptr,
	}

	fmt.Println(jp)

}
