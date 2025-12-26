package main

import (
	"errors"
	"os"
	"os/signal"
	"syscall"

	"context"
	"sync"

	"github.com/uralm1/nxs-backup/ctx"
	"github.com/uralm1/nxs-backup/misc"
	"github.com/uralm1/nxs-backup/routines/cmd_handler"
	"github.com/uralm1/nxs-backup/routines/notification"
)

func main() {
	var err error
	var appCtx *ctx.Ctx

	c_app, cancel_app := context.WithCancelCause(context.Background())
	wg := &sync.WaitGroup{}

	appCtx, err = ctx.AppCtxInit()
	if err != nil {
		//fmt.Println(err.Error())
		os.Exit(1)
	}

	c_sigh, cf := context.WithCancel(c_app)
	defer cf()
	go handle_signals(c_sigh, wg, cancel_app)

	wg.Add(2)
	c_cmdh, cancel_cmdh := context.WithCancel(c_app)
	defer cancel_cmdh()
	c_notification, cancel_notification := context.WithCancel(c_app)
	defer cancel_notification()
	//cmd_handler thread
	go func() {
		defer wg.Done()
		cmd_handler.Runtime(appCtx, c_cmdh, cancel_app, cancel_notification)
	}()
	//notification thread
	go func() {
		defer wg.Done()
		notification.Runtime(appCtx, c_notification)
	}()

	wg.Wait()
	err = context.Cause(c_app)

	if err != nil {
		switch {
		case errors.Is(err, misc.ErrArgSuccessExit):
			os.Exit(0)
		default:
			os.Exit(1)
		}
	}
}

func handle_signals(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelCauseFunc) {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case <-sc:
			//fmt.Println("signal")
			wg.Add(1)
			cancel(nil)
			wg.Done()
		case <-ctx.Done():
			//fmt.Println("signals goroutine cancelled")
			return
		}
	}
}
