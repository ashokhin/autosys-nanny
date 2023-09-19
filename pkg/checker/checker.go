package checker

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	npf "github.com/ashokhin/autosys-nanny/pkg/file"
)

type Checker struct {
	PropertiesFilePath string
	Config             *CheckerConfig
	ConcurrentWorkers  int
	ForceRestart       bool
	processesList      map[int]*Process
	checkerErrorArray  []*error
	AllErrorsArray     []*error
	hostname           string
	logger             *log.Logger
}

// load YAML file from Checker.PropertiesFilePath into Checker.Config
func (c *Checker) loadYaml() error {
	var err error

	level.Debug(*c.logger).Log("msg", "load YAML file", "value", c.PropertiesFilePath)

	if err := npf.LoadYamlFile(c.PropertiesFilePath, &c.Config, *c.logger); err != nil {
		level.Error(*c.logger).Log("msg", "error loading YAML file",
			"value", c.PropertiesFilePath, "error", err.Error())

		return err
	}

	level.Debug(*c.logger).Log("msg", "YAML loaded")

	return err
}

func (c *Checker) getProcessInfo(workerId int, chProcPath <-chan string, chResult chan<- Process) {

	for procPath := range chProcPath {
		var err error
		var process Process

		level.Debug(*c.logger).Log("msg", "collect data from proc path",
			"worker", workerId, "value", procPath)

		fstat, _ := os.Stat(procPath)
		process.ModTime = fstat.ModTime()

		f, err := os.Open(fmt.Sprintf("%s/status", procPath))

		if err != nil {
			level.Warn(*c.logger).Log("msg", "process disappeared",
				"worker", workerId, "value", procPath, "error", err.Error())

			continue
		}

		scanner := bufio.NewScanner(f)

		for scanner.Scan() {
			line := scanner.Text()

			switch {
			case strings.HasPrefix(line, "Name:"):
				processCmd, _ := strings.CutPrefix(line, "Name:")
				process.Cmd = strings.Trim(processCmd, "\t ")
			case strings.HasPrefix(line, "Pid:"):
				processPidStr, _ := strings.CutPrefix(line, "Pid:")

				if process.Pid, err = strconv.Atoi(strings.Trim(processPidStr, "\t ")); err != nil {
					level.Error(*c.logger).Log("msg", "can't convert Pid string to Int",
						"worker", workerId, "value", processPidStr, "error", err.Error())
				}
			case strings.HasPrefix(line, "PPid:"):
				processPPidStr, _ := strings.CutPrefix(line, "PPid:")

				if process.PPid, err = strconv.Atoi(strings.Trim(processPPidStr, "\t ")); err != nil {
					level.Error(*c.logger).Log("msg", "can't convert PPid string to Int",
						"worker", workerId, "value", processPPidStr, "error", err.Error())
				}
			}
		}

		f.Close()

		// get cmdline string
		cmdLineBytes, err := os.ReadFile(fmt.Sprintf("%s/cmdline", procPath))

		if err != nil {
			level.Debug(*c.logger).Log("msg", "process disappeared",
				"worker", workerId, "value", procPath, "error", err.Error())

			continue
		}

		// replace 'null' (\u0000) byte by 'space' (\u0020)
		cmdlineString := strings.Replace(string(cmdLineBytes), "\u0000", " ", -1)
		process.Cmdline = strings.TrimRight(cmdlineString, "\t ")

		level.Debug(*c.logger).Log("msg", "process info",
			"worker", workerId, "value", fmt.Sprintf("%+v", process))

		chResult <- process
	}
}

func (c *Checker) getProcessesList() {
	// init processes map
	c.processesList = make(map[int]*Process)
	// search proc paths with PIDs
	matches, _ := filepath.Glob("/proc/[0-9]*")

	// don't need to start goroutines more than processes found
	if len(matches) < c.ConcurrentWorkers {
		c.ConcurrentWorkers = len(matches)
	}

	chProcPath := make(chan string, len(matches))
	chResult := make(chan Process, len(matches))

	level.Debug(*c.logger).Log("msg", "get processes list")
	level.Debug(*c.logger).Log("msg", "start proc workers",
		"value", c.ConcurrentWorkers)
	// run N worker goroutines for concurrent processing files in /proc/*
	for i := 1; i <= c.ConcurrentWorkers; i++ {
		go c.getProcessInfo(i, chProcPath, chResult)
	}

	for _, procPath := range matches {
		chProcPath <- procPath
	}

	// close channel after write all filepath strings into channel
	close(chProcPath)

	for i := 0; i < len(matches); i++ {
		process := <-chResult

		if process.Pid != 0 {
			c.processesList[process.Pid] = &process
		}
	}

	if len(matches) != len(c.processesList) {
		level.Warn(*c.logger).Log("msg", "len(matches) != len(c.processes)",
			"matches", len(matches), "processes", len(c.processesList))
	}
}

// if service pid found than return `true` otherwise return `false`
func (c *Checker) searchServicePid(service *Service) bool {
	level.Debug(*c.logger).Log("msg", "search service pid",
		"value", service.ProcessName)

	for pid, p := range c.processesList {

		if strings.Contains(p.Cmdline, service.ProcessName) {
			level.Debug(*c.logger).Log("msg", "service pid found in process list",
				"service", service.ProcessName, "value", pid)
			service.process = p

			return true
		}
	}

	return false
}

func (c *Checker) checkService(service *Service, wg *sync.WaitGroup) error {
	var err error
	defer wg.Done()

	if len(service.ProcessName) == 0 {
		level.Warn(*c.logger).Log("msg", "'process_name' should contain value")

		return &ErrNoProcName{}
	}

	level.Debug(*c.logger).Log("msg", "processing service",
		"value", service.ProcessName)

	if c.searchServicePid(service) {

		return nil
	}

	if !service.Disabled {
		level.Warn(*c.logger).Log("msg", "service not found in process list",
			"value", service.ProcessName)
	}

	return err
}

func (c *Checker) collectData() error {
	var err error
	var wg sync.WaitGroup

	// load YAML file from Checker.PropertiesFilePath into Checker.Config
	if err = c.loadYaml(); err != nil {

		return err
	}

	if c.hostname, err = os.Hostname(); err != nil {

		return err
	}

	c.getProcessesList()

	for _, service := range c.Config.Services {
		wg.Add(1)
		level.Debug(*c.logger).Log("msg", "run service checks",
			"value", service.ProcessName)

		go c.checkService(service, &wg)
	}

	wg.Wait()

	return err
}

func (c *Checker) NewLogger(logger *log.Logger) {
	c.logger = logger
}

func (c *Checker) List() {
	if err := c.collectData(); err != nil {
		level.Error(*c.logger).Log("msg", "got error when try to collect data",
			"error", err.Error())

		os.Exit(1)
	}

	// create tabWriter output filter
	w := tabwriter.NewWriter(os.Stdout, 4, 4, 4, ' ', tabwriter.TabIndent|tabwriter.Debug)
	fmt.Fprintln(w, "Service\tRunning\tDisabled\tPID\tStartTime\tUptime\tCmdLine")
	for _, s := range c.Config.Services {
		s.Logger = c.logger

		// skip empty service
		if len(s.ProcessName) == 0 {
			continue
		}

		if s.process != nil {
			fmt.Fprintf(w, "%s\t%t\t%t\t%d\t%s\t%s\t%s\n", s.ProcessName,
				(s.process != nil), s.Disabled, s.process.Pid, s.process.ModTime,
				time.Since(s.process.ModTime), s.process.Cmdline)
		} else {
			fmt.Fprintf(w, "%s\t%t\t%t\t%d\t%s\t%s\t%s\n", s.ProcessName,
				(s.process != nil), s.Disabled, 0, "null", "null", "null")
		}
	}
	w.Flush()
}

func (c *Checker) CheckAndRestart() {

	if err := c.collectData(); err != nil {
		c.checkerErrorArray = append(c.checkerErrorArray, &err)

		return
	}

	for _, service := range c.Config.Services {
		service.Logger = c.logger

		// skip empty service
		if len(service.ProcessName) == 0 {
			continue
		}

		if (service.process == nil) || c.ForceRestart {

			service.RestartProcess(c.ForceRestart)
		}

		if (service.process != nil) && (service.Disabled) {

			service.RestartProcess(c.ForceRestart)
		}
	}
}

func (c *Checker) SendEmail() bool {
	var gotErrors bool

	c.Config.Mailer.Logger = c.logger
	// save mailing_list form YAML key 'general.mailing_list' before processing services
	c.Config.to = c.Config.Mailer.Headers.To

	for _, s := range c.Config.Services {

		if len(s.errorArray) > 0 {
			gotErrors = true
			for _, e := range s.errorArray {
				level.Error(*c.logger).Log("msg", "service got errors", "service", s.ProcessName, "error", e)
			}
			// add service's errors to global array
			c.AllErrorsArray = append(c.AllErrorsArray, s.errorArray...)
			c.Config.Mailer.Headers.To = s.MailList
			c.Config.Mailer.Headers.Subject = fmt.Sprintf("%s | %s restarted", strings.ToUpper(c.hostname), s.ProcessName)
			if err := c.Config.Mailer.SendHtmlEmail(s.ProcessName, "service", s.errorArray); err != nil {
				c.AllErrorsArray = append(c.AllErrorsArray, &err)
			}
		}
	}

	if len(c.checkerErrorArray) > 0 {
		gotErrors = true

		for _, e := range c.checkerErrorArray {
			level.Error(*c.logger).Log("msg", "checker got errors", "error", e)
			c.AllErrorsArray = append(c.AllErrorsArray, e)
		}
	}

	if len(c.AllErrorsArray) > 0 {
		// turn back mailing_list form YAML key 'general.mailing_list' after processing services
		c.Config.Mailer.Headers.To = c.Config.to
		c.Config.Mailer.Headers.Subject = fmt.Sprintf("%s | Nanny script got errors", strings.ToUpper(c.hostname))
		if err := c.Config.Mailer.SendHtmlEmail("Nanny", "script", c.AllErrorsArray); err != nil {
			c.AllErrorsArray = append(c.AllErrorsArray, &err)
		}
	}

	return gotErrors
}
