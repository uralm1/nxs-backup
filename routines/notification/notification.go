// this file was modified as of a derivative work of nxs-backup

package notification

import (
	"context"
	"strings"

	"github.com/uralm1/nxs-backup/ctx"
	"github.com/uralm1/nxs-backup/interfaces"
	"github.com/uralm1/nxs-backup/modules/logger"
)

// Runtime executes the routine
func Runtime(cc *ctx.Ctx, rctx context.Context) error {
	cc.Log.Trace("notification routine: start")

	for {
		select {
		case event := <-cc.EventCh:
			job, flush_flag := strings.CutSuffix(event.JobName, "_flush_notification")
			if flush_flag {
				event.JobName = job
			}

			logger.WriteLog(cc.Log, event)

			for _, n := range cc.Notifiers {
				n.TakeEvent(cc.Log, event)

				if flush_flag || !n.CanCombineMessages() {
					flushNotifier(cc, n)
				}
			}
		case <-rctx.Done():
			flushBufferedNotifiers(cc)
			cc.EventsWG.Wait()
			cc.Log.Trace("notification routine: done")
			return nil
		}
	}
}

func flushNotifier(cc *ctx.Ctx, n interfaces.Notifier) {
	cc.EventsWG.Add(1)
	go func(n interfaces.Notifier) {
		defer cc.EventsWG.Done()
		n.SendBuffer(cc.Log)
	}(n)
}

func flushBufferedNotifiers(cc *ctx.Ctx) {
	for _, n := range cc.Notifiers {
		if n.CanCombineMessages() {
			flushNotifier(cc, n)
		}
	}
}
