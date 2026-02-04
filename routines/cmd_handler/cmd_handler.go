// this file was modified as of a derivative work of nxs-backup

package cmd_handler

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/uralm1/nxs-backup/ctx"
)

func Runtime(cc *ctx.Ctx, rctx context.Context, cancel_app context.CancelCauseFunc, cancel_notification context.CancelFunc) error {
	var err error
	var wg sync.WaitGroup

	cc.Log.Trace("cmd routine: start")

	wg.Add(1)
	go func() {
		defer wg.Done()
		cc.Cmd.Run()
	}()

	for {
		select {
		case <-rctx.Done():
			wg.Wait()
			cc.Log.Trace("cmd routine: shutdown")
			return nil
		case err = <-cc.Done:
			wg.Wait()
			if err != nil {
				cc.Log.WithFields(logrus.Fields{"details": err}).Errorf("Cmd routine fail:")
				cancel_app(err)
				return err
			}
			cc.Log.Trace("cmd routine: done")
			cancel_notification() //app.RoutineShutdown("notification")
			return err
		}
	}
}
