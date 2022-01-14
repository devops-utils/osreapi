//go:build !windows

package main

import (
    "context"
    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
    "github.com/xmapst/osreapi"
    "github.com/xmapst/osreapi/cache"
    _ "github.com/xmapst/osreapi/cmd"
    "github.com/xmapst/osreapi/engine"
    "github.com/xmapst/osreapi/routers"
    "github.com/xmapst/osreapi/utils"
    "gopkg.in/alecthomas/kingpin.v2"
    "io/ioutil"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

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
	// clear old script
	utils.ClearTmpDirOldScript()
	// create tmp db
    cache.Cache = cache.NewExpiredMap()
	engine.NewExecPool(utils.PoolSize)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)

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
		if err := srv.ListenAndServe(); err != nil {
			logrus.Error(err)
		}
	}()
	logrus.Info("server is running ...")

	<-signals
	logrus.Info("shutdown server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	_ = srv.Shutdown(ctx)
	cache.Cache.Close()
}
