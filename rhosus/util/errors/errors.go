package errors

import "errors"

var (
	ErrEntryNotFound      = errors.New("entry not found")
	ErrEntryAlreadyExists = errors.New("entry already exists")
)

func ErrorResponse() {

}
