package notification

import (
	"context"

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
			logger.WriteLog(cc.Log, event)
			for _, n := range cc.Notifiers {
				cc.EventsWG.Add(1)
				go func(n interfaces.Notifier) {
					n.Send(cc.Log, event)
					cc.EventsWG.Done()
				}(n)
			}
		case <-rctx.Done():
			cc.EventsWG.Wait()
			cc.Log.Trace("notification routine: done")
			return nil
		}
	}
}
