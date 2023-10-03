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
	"github.com/ashokhin/autosys-nanny/pkg/mailer"
)

type Checker struct {
	PropertiesFilePath string
	Config             *CheckerConfig
	ConcurrentWorkers  int
	ForceRestart       bool
	checkerErrorArray  []*error
	AllErrorsArray     []*error
	hostname           string
	logger             *log.Logger
}

var processesList map[int]*Process

func (c *Checker) String() string {
	return fmt.Sprintf("%+v", *c)
}

// load YAML file from Checker.PropertiesFilePath into Checker.Config
func (c *Checker) loadYaml() error {
	var err error

	level.Debug(*c.logger).Log("msg", "load yaml file", "value", c.PropertiesFilePath)

	if err := npf.LoadYamlFile(c.PropertiesFilePath, &c.Config, *c.logger); err != nil {
		level.Error(*c.logger).Log("msg", "error loading yaml file",
			"value", c.PropertiesFilePath, "error", err.Error())

		return err
	}

	if c.Config.Mailer != nil {
		c.Config.Mailer.SafeStorePassword()
	}

	level.Debug(*c.logger).Log("msg", "yaml loaded")

	return err
}

func (c *Checker) getProcessInfo(workerId int, chProcPath <-chan string, chResult chan<- Process) {

	for procPath := range chProcPath {
		var err error
		var process Process

		level.Debug(*c.logger).Log("msg", "collect data from proc path",
			"worker", workerId, "value", procPath)

		fstat, err := os.Stat(procPath)

		if err != nil {
			level.Warn(*c.logger).Log("msg", "process disappeared",
				"worker", workerId, "value", procPath, "error", err.Error())

			continue
		}

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
					level.Error(*c.logger).Log("msg", "can't convert pid string to int",
						"worker", workerId, "value", processPidStr, "error", err.Error())
				}
			case strings.HasPrefix(line, "PPid:"):
				processPPidStr, _ := strings.CutPrefix(line, "PPid:")

				if process.PPid, err = strconv.Atoi(strings.Trim(processPPidStr, "\t ")); err != nil {
					level.Error(*c.logger).Log("msg", "can't convert ppid string to Int",
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

		// replace 'null' (\u0000) UTF-8 symbol by 'space' (" ")
		cmdlineString := strings.Replace(string(cmdLineBytes), "\u0000", " ", -1)
		process.Cmdline = strings.TrimRight(cmdlineString, "\t ")

		level.Debug(*c.logger).Log("msg", "process info",
			"worker", workerId, "value", fmt.Sprintf("%+v", process))

		chResult <- process
	}
}

func (c *Checker) getProcessesList() {
	// init processes map
	processesList = make(map[int]*Process)
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
			processesList[process.Pid] = &process
		}
	}

	if len(matches) != len(processesList) {
		level.Warn(*c.logger).Log("msg", "len(matches) != len(c.processes)",
			"matches", len(matches), "processes", len(processesList))
	}
}

// if service pid found than return `true` otherwise return `false`
func (c *Checker) searchServicePid(service *Service) bool {
	level.Debug(*c.logger).Log("msg", "search service pid",
		"value", service.ProcessName)

	for pid, p := range processesList {

		if strings.Contains(p.Cmdline, service.ProcessName) {
			level.Debug(*c.logger).Log("msg", "service pid found in process list",
				"service", service.ProcessName, "value", pid)

			if service.process != nil {
				// search main pid (always lower number in process tree)
				if service.process.Pid > p.Pid {
					service.process = p
				}
			} else {
				service.process = p
			}
		}
	}

	return service.process != nil
}

func (c *Checker) checkService(service *Service, wg *sync.WaitGroup) error {
	var err error
	defer wg.Done()

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

	for sliceIndex, service := range c.Config.Services {
		level.Debug(*c.logger).Log("msg", "run service checks",
			"value", service.ProcessName)

		if len(service.ProcessName) == 0 {
			procErr := fmt.Errorf("'Nanny' script error: services_list[%d].process_name should contain value", sliceIndex)
			c.checkerErrorArray = append(c.checkerErrorArray, &procErr)

			level.Error(*c.logger).Log("msg", "error load process details from yaml",
				"error", procErr.Error())

			continue
		}

		wg.Add(1)

		go c.checkService(service, &wg)
	}

	wg.Wait()

	return err
}

func (c *Checker) NewLogger(logger *log.Logger) {
	c.logger = logger
}

func (c *Checker) List() error {
	if err := c.collectData(); err != nil {
		level.Warn(*c.logger).Log("msg", "got error when try to collect data",
			"error", err.Error())

		err1 := fmt.Errorf("'Nanny' script error: %s", err.Error())

		c.checkerErrorArray = append(c.checkerErrorArray, &err1)

		return err
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

	return nil
}

func (c *Checker) CheckAndRestart() error {

	if err := c.collectData(); err != nil {
		err1 := fmt.Errorf("'Nanny' script error: %s", err.Error())
		c.checkerErrorArray = append(c.checkerErrorArray, &err1)

		return err
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

	return nil
}

func (c *Checker) ReportErrors() bool {
	var gotErrors bool

	subjectPrefix := strings.ToUpper(c.hostname)

	if c.Config.Mailer != nil {
		c.Config.Mailer.Logger = c.logger

		if c.Config.Mailer.Headers != nil {
			// save mailing_list form YAML key 'general.mailing_list' before processing services
			c.Config.to = c.Config.Mailer.Headers.To
		}

		if len(c.Config.Mailer.SubjectPrefix) > 0 {
			subjectPrefix = c.Config.Mailer.SubjectPrefix
		}
	} else {
		c.Config.Mailer = new(mailer.Mailer)
	}

	for _, s := range c.Config.Services {

		// report about errors in services
		if len(s.errorArray) > 0 {
			gotErrors = true
			for _, e := range s.errorArray {
				level.Error(*c.logger).Log("msg", "service got errors", "service", s.ProcessName, "error", e)
			}
			// add service's errors to global array
			c.AllErrorsArray = append(c.AllErrorsArray, s.errorArray...)

			if len(s.MailList) == 0 {
				level.Debug(*c.logger).Log("msg", "service doesn't have 'mailing_list'. skip sending emails",
					"service", s.ProcessName)

				continue
			}

			if err := c.Config.Mailer.CheckSettings(); err != nil {
				level.Warn(*c.logger).Log("msg", "checker mail config inconsistent. skip sending emails",
					"service", s.ProcessName, "error", err)

				c.AllErrorsArray = append(c.AllErrorsArray, &err)

				continue
			}

			c.Config.Mailer.Headers.To = s.MailList

			c.Config.Mailer.Headers.Subject = fmt.Sprintf("%s | '%s' alert - restarted", subjectPrefix, s.ProcessName)

			if err := c.Config.Mailer.SendHtmlEmail(s.errorArray); err != nil {
				c.AllErrorsArray = append(c.AllErrorsArray, &err)
			}
		}
	}

	if len(c.checkerErrorArray) > 0 {
		gotErrors = true

		for _, e := range c.checkerErrorArray {
			level.Error(*c.logger).Log("msg", "nanny script got errors", "error", e)
			c.AllErrorsArray = append(c.AllErrorsArray, e)
		}

		if len(c.Config.to) == 0 {
			level.Debug(*c.logger).Log("msg", "nanny script doesn't have 'mailing_list'.  skip sending emails")

			return gotErrors
		}

		if err := c.Config.Mailer.CheckSettings(); err != nil {
			level.Warn(*c.logger).Log("msg", "nanny script mail config inconsistent. skip sending emails",
				"error", err)

			c.AllErrorsArray = append(c.AllErrorsArray, &err)

			return gotErrors
		}

		// switch back to global 'mailing_list'
		c.Config.Mailer.Headers.To = c.Config.to
		c.Config.Mailer.Headers.Subject = fmt.Sprintf("%s | Nanny script got errors", subjectPrefix)

		if err := c.Config.Mailer.SendHtmlEmail(c.checkerErrorArray); err != nil {
			c.AllErrorsArray = append(c.AllErrorsArray, &err)
		}
	}

	return gotErrors
}
