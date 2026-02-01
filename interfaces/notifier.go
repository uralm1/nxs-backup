// this file was modified as of a derivative work of nxs-backup

package interfaces

import (
	"github.com/sirupsen/logrus"

	"github.com/uralm1/nxs-backup/modules/logger"
)

type Notifier interface {
	CanCombineMessages() bool
	ClearBuffer()
	TakeEvent(log *logrus.Logger, rec logger.LogRecord)
	SendBuffer(log *logrus.Logger)
}
