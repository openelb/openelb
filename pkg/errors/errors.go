package errors

const (
	UnknowError = iota
	ParaInvalidError
	EIPDuplicateError
	EIPIsUsedError
	EIPNotExist //assign or find
	EIPNotEnoughError
)

type PorterError struct {
	Code    int
	Message string
}

func (e PorterError) Error() string {
	return e.Message
}

func ReasonForError(err error) int {
	switch t := err.(type) {
	case PorterError:
		return t.Code
	}
	return UnknowError
}

func IsParaInvalidError(err error) bool {
	if ReasonForError(err) == ParaInvalidError {
		return true
	}

	return false
}
