//go:build windows

package main

import (
	"context"
	"fmt"
	"github.com/StackExchange/wmi"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/xmapst/osreapi"
	"github.com/xmapst/osreapi/cache"
	_ "github.com/xmapst/osreapi/cmd"
	"github.com/xmapst/osreapi/engine"
	"github.com/xmapst/osreapi/routers"
	"github.com/xmapst/osreapi/utils"
	"golang.org/x/sys/windows/svc"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var stopCh = make(chan bool)

// @title OSRemoteExecution API
// @version v1.0.0
// @description This is a os remote executor orchestration script interface.
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
func main() {
	kingpin.Version(osreapi.VersionIfo())
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	osreapi.PrintHeadInfo()

	if !utils.DebugEnabled {
		logrus.SetOutput(ioutil.Discard)
	}
	initWbem()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	// clear old script
	utils.ClearTmpDirOldScript()
	// create tmp db
	cache.Cache = cache.NewExpiredMap()
	engine.NewExecPool(utils.PoolSize)

	isWindowsService, err := svc.IsWindowsService()
	if err != nil {
		logrus.Fatalln(err)
	}
	if isWindowsService {
		go func() {
			err = svc.Run(utils.ServiceName, &windowsExporterService{stopCh: stopCh})
			if err != nil {
				logrus.Errorf("failed to start service: %v", err)
				return
			}
		}()
	}

	gin.SetMode(gin.ReleaseMode)
	if utils.DebugEnabled {
		gin.SetMode(gin.DebugMode)
	}
	gin.DisableConsoleColor()
	srv := &http.Server{
		Addr:         utils.ListenAddress,
		WriteTimeout: utils.WebTimeout,
		ReadTimeout:  utils.WebTimeout,
		IdleTimeout:  utils.WebTimeout,
		Handler:      routers.Router(),
	}
	go func() {
		logrus.Info("listenAddress ", utils.ListenAddress)
		if err = srv.ListenAndServe(); err != nil {
			logrus.Error(err)
		}
	}()
	logrus.Info("server is running ...")
	for {
		select {
		case <-signals:
			goto RETURN
		case <-stopCh:
			goto RETURN
		}
	}
RETURN:
	logrus.Info("shutdown server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_ = srv.Shutdown(ctx)
	cache.Cache.Close()
}

func initWbem() {
	// This initialization prevents a memory leak on WMF 5+. See
	// https://github.com/prometheus-community/windows_exporter/issues/77 and
	// linked issues for details.
	logrus.Debug("initializing SWbemServices")
	s, err := wmi.InitializeSWbemServices(wmi.DefaultClient)
	if err != nil {
		logrus.Fatalln(err)
	}
	wmi.DefaultClient.AllowMissingFields = true
	wmi.DefaultClient.SWbemServicesClient = s
}

type windowsExporterService struct {
	stopCh chan<- bool
}

func (s *windowsExporterService) Execute(_ []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				s.stopCh <- true
				break loop
			default:
				logrus.Error(fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}
