package errors

import (
	"errors"
	"fmt"
)

// Prefix is the default error string prefix
const Prefix = "opcua: "

// Errorf wraps fmt.Errorf
func Errorf(format string, a ...interface{}) error {
	return fmt.Errorf(Prefix+format, a...)
}

// New wraps errors.New
func New(text string) error {
	return errors.New(Prefix + text)
}

// Is wraps errors.Is
func Is(err error, target error) bool {
	return errors.Is(err, target)
}

// As wraps errors.As
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Unwrap wraps errors.Unwrap
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// Join wraps errors.Join
func Join(errs ...error) error {
	return errors.Join(errs...)
}

// Equal returns true if the two errors have the same error message.
//
// todo(fs): the reason we need this function and cannot just use
// todo(fs): reflect.DeepEqual(err1, err2) is that by using github.com/pkg/errors
// todo(fs): the underlying stack traces change and because of this the errors
// todo(fs): are no longer comparable. This is a downside of basing our errors
// todo(fs): errors implementation on github.com/pkg/errors and we may want to
// todo(fs): revisit this.
// todo(fs): See https://play.golang.org/p/1WqB7u4BUf7 (by @kung-foo)
func Equal(err1, err2 error) bool {
	if err1 == nil && err2 == nil {
		return true
	}
	if err1 != nil && err2 != nil {
		return err1.Error() == err2.Error()
	}
	return false
}
