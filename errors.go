// Package errors implements functions to deal with error handling.
package errors

import (
	"fmt"
	"runtime"
	"strconv"
)

const _MAX_FRAMES = 100

type frame struct {
	file     string
	line     int
	function string
}

// Error represent an error with traceback.
type Error struct {
	kind              string
	message           string
	cause             error
	extra             map[string]any
	stackTrace        []frame
	captureStackTrace bool
}

// New creates a new Error with a message (can have a format) and
// sets to optionally capture the stack trace when raised (default is true).
func New(message string, captureStackTrace ...bool) Error {
	_captureStackTrace := true
	if len(captureStackTrace) > 0 {
		_captureStackTrace = captureStackTrace[0]
	}

	return Error{
		kind:              message,
		message:           message,
		cause:             nil,
		extra:             nil,
		stackTrace:        nil,
		captureStackTrace: _captureStackTrace,
	}
}

// Raise creates a new Error instance formatting its message if
// needed and optionally captures its stack trace.
func (self Error) Raise(args ...any) *Error {
	var stackTrace []frame

	if self.captureStackTrace {
		stackFrames := make([]uintptr, _MAX_FRAMES)

		length := runtime.Callers(2, stackFrames)
		if length > 0 {
			stackTrace = make([]frame, 0, length)

			cframes := runtime.CallersFrames(stackFrames[:length])
			for {
				cframe, more := cframes.Next()
				stackTrace = append(stackTrace, frame{
					file:     cframe.File,
					line:     cframe.Line,
					function: cframe.Function,
				})

				if !more {
					break
				}
			}
		}
	}

	return &Error{
		kind:       self.kind,
		message:    fmt.Sprintf(self.message, args...),
		cause:      nil,
		extra:      make(map[string]any),
		stackTrace: stackTrace,
	}
}

// Skip removes n frames of the raised Error.
func (self *Error) Skip(frames int) *Error {
	if self.captureStackTrace {
		self.stackTrace = self.stackTrace[frames:]
	}

	return self
}

// With adds more context to the raised Error's message.
func (self *Error) With(message string, args ...any) *Error {
	self.message += ": " + fmt.Sprintf(message, args...)

	return self
}

// Extra adds extra information to the raised Error.
func (self *Error) Extra(extra map[string]any) *Error {
	for key, value := range extra {
		self.extra[key] = value
	}

	return self
}

// Cause wraps an error into the raised Error.
func (self *Error) Cause(err error) *Error {
	self.cause = err

	return self
}

// Is compares whether an error is Error's type.
func (self Error) Is(err error) bool {
	if err == nil {
		return false
	}

	switch other := err.(type) {
	case Error:
		return self.kind == other.kind
	case *Error:
		return self.kind == other.kind
	}

	return false
}

// String implements the Stringer interface.
func (self Error) String() string {
	causeMessage := ""
	if self.cause != nil {
		causeMessage = ": " + self.cause.Error()
	}

	return self.message + causeMessage
}

// Error implements the Error interface.
func (self Error) Error() string {
	return self.String()
}

// MarshalText implements the TextMarshaler interface.
func (self Error) MarshalText() ([]byte, error) {
	return []byte(self.String()), nil
}

// MarshalJSON implements the JSONMarshaler interface.
func (self Error) MarshalJSON() ([]byte, error) {
	return []byte("\"" + self.String() + "\""), nil
}

func (self Error) report(all bool, seenTraces map[string]bool) string {
	if seenTraces == nil {
		seenTraces = make(map[string]bool)
	}

	report := ""

	if len(self.stackTrace) > 0 {
		ellipsis := false

		for i := len(self.stackTrace) - 1; i >= 0; i-- {
			fileline := self.stackTrace[i].file + ":" + strconv.Itoa(self.stackTrace[i].line)

			_, seen := seenTraces[fileline]
			if !seen {
				seenTraces[fileline] = true
				report += "    " + fileline + "\n"
				report += "        " + self.stackTrace[i].function + "\n"
			} else if !ellipsis {
				ellipsis = true
				report += "    [...]\n"
			}
		}
	} else {
		report += "    (Stack trace not available)\n"
	}

	report += "\x1b[0;31m" + self.message + "\x1b[0m\n"

	if len(self.extra) > 0 {
		report += "    "
		for key, value := range self.extra {
			report += key + "=" + fmt.Sprintf("%v", value) + " "
		}
		report += "\n"
	}

	if all && self.cause != nil {
		report += "\nCaused by the following error:\n"
		switch cause := self.cause.(type) {
		case Error:
			report += cause.report(all, seenTraces)
		case *Error:
			report += cause.report(all, seenTraces)
		default:
			report += "    (Stack trace not available)\n"
			report += "\x1b[0;31m" + cause.Error() + "\x1b[0m\n"
		}
	}

	return report
}

// Report returns a string containing all the information about the first
// error (including the message, stack trace, extra...) or about all errors
// wrapped within the Error itself (default is all).
func (self Error) Report(all ...bool) string {
	_all := true
	if len(all) > 0 {
		_all = all[0]
	}

	report := "\x1b[1;91m" + self.String() + "\x1b[0m\n\n"
	report += "Traceback (most recent call last):\n"
	report += self.report(_all, nil)

	return report
}

// Format implements the Formatter interface:
// - %s: Error message
// - %v: First error report
// - %+v: All errors reports
// - default: Error message
func (self Error) Format(format fmt.State, verb rune) {
	switch verb {
	case 's':
		format.Write([]byte(self.String()))
	case 'v':
		if format.Flag('+') {
			format.Write([]byte(self.Report()))
		} else {
			format.Write([]byte(self.Report(false)))
		}
	default:
		format.Write([]byte(self.String()))
	}
}
