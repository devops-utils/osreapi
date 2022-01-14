package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/xmapst/osreapi/utils"
	"gopkg.in/alecthomas/kingpin.v2"
)

func init() {
	// flags
	kingpin.Flag(
		"addr",
		`host:port for execution.`,
	).Default(":2376").StringVar(&utils.ListenAddress)
	kingpin.Flag(
		"debug",
		`Enable debug messages`,
	).Default("false").BoolVar(&utils.DebugEnabled)
	kingpin.Flag(
		"exec_timeout",
		`Set the default exec command expire time. Example: "exec_timeout=30m"`,
	).Default("30m").DurationVar(&utils.ExecTimeOut)
	kingpin.Flag(
		"timeout",
		`Timeout for calling endpoints on the engine`,
	).Default("30s").DurationVar(&utils.WebTimeout)
	kingpin.Flag(
		"max-requests",
		`Maximum number of concurrent requests. 0 to disable.`,
	).Default("0").Int64Var(&utils.MaxRequests)
	kingpin.Flag(
		"pool_size",
		`Set the size of the execution work pool. Example: "exec_timeout=30m"`,
	).Default("300").IntVar(&utils.PoolSize)
	_ = utils.EnsureDirExist(utils.TmpDir)
	// log format init
	logrus.SetReportCaller(true)
	logrus.SetLevel(logrus.DebugLevel)
	logrus.AddHook(utils.NewRotateHook())
	logrus.SetFormatter(&utils.FileFormatter{})
}
