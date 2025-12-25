package cmd_handler

import (
	"github.com/sirupsen/logrus"

	"github.com/uralm1/nxs-backup/appctx"

	"github.com/uralm1/nxs-backup/ctx"
)

func Runtime(app appctx.App) error {
	var err error

	cc := app.ValueGet().(*ctx.Ctx)

	cc.Log.Trace("cmd routine: start")
	go cc.Cmd.Run()

	for {
		select {
		case <-app.SelfCtxDone():
			cc.Log.Trace("cmd routine: shutdown")
			return nil
		case err = <-cc.Done:
			if err != nil {
				cc.Log.WithFields(logrus.Fields{"details": err}).Errorf("cmd routine fail:")
				app.Shutdown(err)
				return err
			}
			cc.Log.Trace("cmd routine: done")
			app.RoutineShutdown("notification")
			return err
		}
	}
}
