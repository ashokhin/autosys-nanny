package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	chk "github.com/ashokhin/autosys-nanny/pkg/checker"
)

var (
	checker           chk.Checker
	appName           = "autosys-nanny"
	appBranch         = "None"
	appVersion        = "dev"
	appRevision       = "0"
	appOrigin         = "./"
	appBuildUser      = "nobody"
	appBuildDate      = "None"
	app               = kingpin.New("autosys-nanny", "A command-line tool for managing services defined in yaml configuration file.")
	concurrentWorkers = app.Flag("workers-num", "Maximum number of concurrent workers for processing services").Short('w').Default("100").Int()
	debug             = app.Flag("debug", "Enable debug mode.").Short('v').Default("false").Bool()
	forceRestart      = app.Flag("force-restart", "Restart services even than they already running").Short('r').Default("false").Bool()
	listOnly          = app.Flag("list", "Only check services without restart and list them").Short('l').Default("false").Bool()
	propertyFile      = app.Flag("config", "YAML file with services properties.").Short('c').Default("./services.yaml").String()
	logFile           = app.Flag("log-file", "Path to log file").Short('f').Default("").String()
	supported_os      = []string{"linux"}
	logger            log.Logger
)

func printVersion() string {
	return fmt.Sprintf(`%q build info:
	version:              %q
	repo:                 %q
	branch:               %q
	revision:             %q
	build_user:           %q
	build_date:           %q`, appName, appVersion, appOrigin, appBranch, appRevision, appBuildUser, appBuildDate)
}

func checkOS() error {

	if !slices.Contains(supported_os, runtime.GOOS) {
		return fmt.Errorf("os %s unsupported", runtime.GOOS)
	}

	return nil
}

func init() {

	if err := checkOS(); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}

	app.Version(printVersion())
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if len(*logFile) == 0 {
		logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	} else {
		logFileWriter, err := os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			panic(fmt.Errorf("error open log file '%s'.\nerror: '%s'", *logFile, err.Error()))
		} else {
			logger = log.NewLogfmtLogger(log.NewSyncWriter(logFileWriter))
		}

	}

	if *debug {
		logger = level.NewFilter(logger, level.AllowDebug())
	} else {
		logger = level.NewFilter(logger, level.AllowInfo())
	}

	/*
		timestampFormat := log.TimestampFormat(
			func() time.Time { return time.Now().UTC() },
			"2006-01-02T15:04:05.000000Z07:00",
		)
		logger = log.With(logger, "timestamp", timestampFormat, "caller", log.DefaultCaller)
	*/

	logger = log.With(logger, "timestamp", log.DefaultTimestamp, "caller", log.DefaultCaller)
	checker.NewLogger(&logger)
	checker.ConcurrentWorkers = *concurrentWorkers
	checker.ForceRestart = *forceRestart
}

func main() {
	timeStart := time.Now()

	checker.PropertiesFilePath, _ = filepath.Abs(*propertyFile)
	if *listOnly {
		checker.List()

		level.Debug(logger).Log("msg", "list success", "elapsed_time", time.Since(timeStart))
		return
	}

	checker.CheckAndRestart()
	if checker.ReportErrors() {
		level.Warn(logger).Log("msg", "checks completed with errors",
			"elapsed_time", time.Since(timeStart))

		for _, e := range checker.AllErrorsArray {
			level.Warn(logger).Log("msg", "error details", "error", e)
		}

		os.Exit(1)
	}

	level.Info(logger).Log("msg", "checks success", "elapsed_time", time.Since(timeStart))

	os.Exit(0)
}
