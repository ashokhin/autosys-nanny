package file

// directory error
type ErrIsDir struct {
	message string
}

func newIsDirError(message string) *ErrIsDir {
	return &ErrIsDir{
		message: message,
	}
}

func (e *ErrIsDir) Error() string {
	return e.message
}
