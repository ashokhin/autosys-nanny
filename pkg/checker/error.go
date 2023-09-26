package checker

import (
	"fmt"
)

type ErrNoProcName struct{}

func (e *ErrNoProcName) Error() string {
	return "no value for property 'process_name'"
}

type ErrZeroPid struct {
	service string
}

func (e *ErrZeroPid) Error() string {
	return fmt.Sprintf("service '%s' stop failed. service process pid = 0. looks like service has already stopped", e.service)
}

type ErrNoStartCmd struct {
	service string
}

func (e *ErrNoStartCmd) Error() string {
	return fmt.Sprintf("service '%s' start failed. service doesn't have start command in 'start_cmd' property", e.service)
}

type ErrNoPidFile struct {
	service string
}

func (e *ErrNoPidFile) Error() string {
	return fmt.Sprintf("service '%s' doesn't have pid file which path defined in 'pid_file' property", e.service)
}

type ErrSrvRestartedForce struct {
	service string
}

func (e *ErrSrvRestartedForce) Error() string {
	return fmt.Sprintf("service '%s' restarted with key --force-restart", e.service)
}
