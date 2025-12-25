package interfaces

import (
	"github.com/sirupsen/logrus"

	"github.com/uralm1/nxs-backup/modules/logger"
)

type Notifier interface {
	Send(log *logrus.Logger, rec logger.LogRecord)
}
