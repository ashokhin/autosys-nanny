package checker

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type Service struct {
	ProcessName  string   `yaml:"process_name"`
	Description  string   `yaml:"description"`
	Disabled     bool     `yaml:"disabled"`
	StartCmd     string   `yaml:"start_cmd"`
	StopCmd      string   `yaml:"stop_cmd"`
	WorkingDir   string   `yaml:"working_directory"`
	PidFile      string   `yaml:"pid_file"`
	EnvList      []string `yaml:"env_vars"`
	MailList     []string `yaml:"mailing_list"`
	forceRestart bool
	errorArray   []*error
	process      *Process
	Logger       *log.Logger
}

type Process struct {
	Cmd     string
	Cmdline string
	Pid     int
	PPid    int
	ModTime time.Time
}

func (s *Service) deletePidFile() {
	var err error

	if len(s.PidFile) == 0 {
		level.Debug(*s.Logger).Log("msg", "service doesn't have PID file path in 'pid_file' property",
			"value", s.ProcessName)

		return
	}

	level.Debug(*s.Logger).Log("msg", "search PID file for service", "value", s.ProcessName)

	matches, err := filepath.Glob(fmt.Sprintf("./%s", s.PidFile))

	if err != nil {
		level.Warn(*s.Logger).Log("msg", "search pid got error", "service", s.ProcessName,
			"value", fmt.Sprintf("./%s", s.PidFile))
		s.errorArray = append(s.errorArray, &err)
	}

	for _, f := range matches {
		level.Debug(*s.Logger).Log("msg", "delete pid file", "service", s.ProcessName,
			"value", f)

		if err := os.Remove(f); err != nil {
			level.Error(*s.Logger).Log("msg", "got error when try to delete PID file",
				"service", s.ProcessName, "value", f, "error", err.Error())

			s.errorArray = append(s.errorArray, &err)
		}
	}

	if s.Disabled {
		return
	}

	if len(matches) == 0 {
		err = &ErrNoPidFile{s.ProcessName}
		s.errorArray = append(s.errorArray, &err)
	}
}

func (s *Service) kill() error {
	var err error

	p, err := os.FindProcess(s.process.Pid)

	if err != nil {
		s.deletePidFile()

		return err
	}

	if err := p.Kill(); err != nil {

		return err
	}

	return err
}

func (s *Service) stop() error {
	var err error

	if s.process == nil {
		s.deletePidFile()

		if s.Disabled {
			level.Debug(*s.Logger).Log("msg", "service disabled and has already stopped", "value", s.ProcessName)
			return nil
		}

		return &ErrZeroPid{s.ProcessName}
	}

	if len(s.StopCmd) > 0 {
		// if stop command present than exec stop command
		level.Info(*s.Logger).Log("msg", "execute stop command for service",
			"service", s.ProcessName, "value", s.StopCmd)

		cmd := exec.Command("bash", "-c", s.StopCmd)

		level.Debug(*s.Logger).Log("msg", "stop command", "value", cmd.String())

		if err := cmd.Run(); err != nil {
			level.Error(*s.Logger).Log("msg", "got error when try to stop command",
				"value", cmd.String(), "error", err.Error())
		}
	} else {
		// else kill process with 'syscall.SIGKILL' signal
		level.Info(*s.Logger).Log("msg", "execute kill process for service",
			"service", s.ProcessName, "value", "syscall.SIGKILL")

		err = s.kill()
	}

	s.deletePidFile()

	return err
}

func (s *Service) start() error {
	var err error

	// if service disabled than skip start process
	if s.Disabled {
		level.Debug(*s.Logger).Log("msg", "service disabled. skip start process",
			"value", s.ProcessName)

		return nil
	}

	if len(s.StartCmd) == 0 {
		level.Debug(*s.Logger).Log("msg", "service doesn't have start command in 'start_cmd' property",
			"value", s.ProcessName)

		return &ErrNoStartCmd{s.ProcessName}
	}

	cmd := exec.Command("bash", "-c", s.StartCmd, "&")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, s.EnvList...)

	level.Info(*s.Logger).Log("msg", "execute start command",
		"service", s.ProcessName, "value", fmt.Sprintf("%+v", cmd.String()))
	level.Debug(*s.Logger).Log("msg", "environment variables",
		"service", s.ProcessName, "value", fmt.Sprintf("%s", cmd.Env))

	if err := cmd.Start(); err != nil {
		level.Error(*s.Logger).Log("msg", "got error when try to start command",
			"service", s.ProcessName, "value", cmd.String(), "error", err.Error())
	}

	if err == nil && s.forceRestart {
		err = &ErrSrvRestartedForce{s.ProcessName}
	}

	return err
}

func (s *Service) RestartProcess(forceRestart bool) error {
	var err error

	s.forceRestart = forceRestart
	cwd, _ := os.Getwd()

	level.Debug(*s.Logger).Log("msg", "restart service", "value", s.ProcessName)

	if s.WorkingDir != "" {
		defer os.Chdir(cwd)
		level.Debug(*s.Logger).Log("msg", "change current working directory",
			"service", s.ProcessName, "value", s.WorkingDir)

		if err := os.Chdir(s.WorkingDir); err != nil {
			level.Error(*s.Logger).Log("msg", "got error when try to change working directory",
				"service", s.ProcessName, "value", s.WorkingDir, "error", err.Error())

			s.errorArray = append(s.errorArray, &err)

			return err
		}
	}

	if err := s.stop(); err != nil {
		var errZeroPid *ErrZeroPid

		if errors.As(err, &errZeroPid) {
			level.Warn(*s.Logger).Log("msg", "service has already stopped",
				"value", s.ProcessName, "error", err.Error())

			s.errorArray = append(s.errorArray, &err)
		} else {
			level.Error(*s.Logger).Log("msg", "got error when try to stop service",
				"value", s.ProcessName, "error", err.Error())

			s.errorArray = append(s.errorArray, &err)
			return err
		}
	}

	if err := s.start(); err != nil {
		level.Error(*s.Logger).Log("msg", "got error when try to start service",
			"value", s.ProcessName, "error", err.Error())

		s.errorArray = append(s.errorArray, &err)
		return err
	}

	return err
}
