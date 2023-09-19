package checker

import (
	"fmt"
)

type ErrNoProcName struct{}

func (e *ErrNoProcName) Error() string {
	return "no value for property 'process_name'"
}

type ErrZeroPid struct {
	message string
}

func (e *ErrZeroPid) Error() string {
	return fmt.Sprintf("Service '%s' stop failed. Service process PID = 0. Looks like service has already stopped", e.message)
}

type ErrNoStartCmd struct {
	message string
}

func (e *ErrNoStartCmd) Error() string {
	return fmt.Sprintf("Service '%s' doesn't have start command in 'start_cmd' property", e.message)
}

type ErrNoPidFile struct {
	message string
}

func (e *ErrNoPidFile) Error() string {
	return fmt.Sprintf("Service '%s' doesn't have PID file which path defined in 'pid_file' property", e.message)
}

type ErrSrvRestartedForce struct {
	message string
}

func (e *ErrSrvRestartedForce) Error() string {
	return fmt.Sprintf("Service '%s' restarted with key --force-restart", e.message)
}
