package mailer

import "fmt"

type ErrBadMailSettings struct {
	message string
}

func (e *ErrBadMailSettings) Error() string {
	return fmt.Sprint(e.message)
}
