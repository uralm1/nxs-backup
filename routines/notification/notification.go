package notification

import (
	"github.com/uralm1/nxs-backup/appctx"

	"github.com/uralm1/nxs-backup/ctx"
	"github.com/uralm1/nxs-backup/interfaces"
	"github.com/uralm1/nxs-backup/modules/logger"
)

// Runtime executes the routine
func Runtime(app appctx.App) error {
	cc := app.ValueGet().(*ctx.Ctx)
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
		case <-app.SelfCtxDone():
			cc.EventsWG.Wait()
			cc.Log.Trace("notification routine: done")
			return nil
		}
	}
}
