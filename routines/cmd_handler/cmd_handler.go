// this file was modified as of a derivative work of nxs-backup

package cmd_handler

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/uralm1/nxs-backup/ctx"
)

func Runtime(cc *ctx.Ctx, rctx context.Context, app_cancel context.CancelCauseFunc, notification_cancel context.CancelFunc) error {
	var err error

	cc.Log.Trace("cmd routine: start")
	go cc.Cmd.Run()

	for {
		select {
		case <-rctx.Done():
			cc.Log.Trace("cmd routine: shutdown")
			return nil
		case err = <-cc.Done:
			if err != nil {
				cc.Log.WithFields(logrus.Fields{"details": err}).Errorf("cmd routine fail:")
				app_cancel(err)
				return err
			}
			cc.Log.Trace("cmd routine: done")
			notification_cancel() //app.RoutineShutdown("notification")
			return err
		}
	}
}
