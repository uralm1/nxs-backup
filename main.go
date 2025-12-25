package main

import (
	"errors"
	"os"
	"syscall"

	"github.com/uralm1/nxs-backup/appctx"

	"github.com/uralm1/nxs-backup/ctx"
	"github.com/uralm1/nxs-backup/misc"
	"github.com/uralm1/nxs-backup/routines/cmd_handler"
	"github.com/uralm1/nxs-backup/routines/notification"
)

func main() {
	err := appctx.Init(nil).
		RoutinesSet(
			map[string]appctx.RoutineParam{
				"handler": {
					Handler: cmd_handler.Runtime,
				},
				"notification": {
					Handler: notification.Runtime,
				},
			},
		).
		ValueInitHandlerSet(ctx.AppCtxInit).
		SignalsSet([]appctx.SignalsParam{
			{
				Signals: []os.Signal{
					syscall.SIGTERM,
					syscall.SIGINT,
				},
				Handler: sigHandlerTerm,
			},
		}).
		Run()
	if err != nil {
		switch {
		case errors.Is(err, misc.ErrArgSuccessExit):
			os.Exit(0)
		default:
			os.Exit(1)
		}
	}
}

func sigHandlerTerm(sig appctx.Signal) {
	sig.Shutdown(nil)
}
