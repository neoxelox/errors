// Package errors implements functions to deal with error handling.
package errors

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/getsentry/sentry-go"
)

const _MAX_FRAMES = 100

var _ANSI_COLOR_PATTERN = regexp.MustCompile(`\x1b\[[0-9;]*m`)

type frame struct {
	file     string
	line     int
	function string
}

// Error represents an error with traceback and additional info.
type Error struct {
	kind              string
	module            string
	message           string
	cause             error
	extra             map[string]any
	stackTrace        []frame
	captureStackTrace bool
	tags              map[string]string
}

// New creates a new Error with a message (can have a format) and
// sets to optionally capture the stack trace when raised (default is true).
func New(message string, captureStackTrace ...bool) Error {
	_captureStackTrace := true
	if len(captureStackTrace) > 0 {
		_captureStackTrace = captureStackTrace[0]
	}

	module := "unknown"
	stackFrames := make([]uintptr, 1)

	length := runtime.Callers(2, stackFrames)
	if length > 0 {
		frame, _ := runtime.CallersFrames(stackFrames[:length]).Next()

		separator := strings.LastIndex(frame.Function, ".")
		if separator >= 0 {
			module = frame.Function[:separator]
		}
	}

	return Error{
		kind:              message,
		module:            module,
		message:           message,
		cause:             nil,
		extra:             nil,
		stackTrace:        nil,
		captureStackTrace: _captureStackTrace,
		tags:              nil,
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
		kind:              self.kind,
		module:            self.module,
		message:           fmt.Sprintf(self.message, args...),
		cause:             nil,
		extra:             make(map[string]any),
		stackTrace:        stackTrace,
		captureStackTrace: self.captureStackTrace,
		tags:              make(map[string]string),
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

// Tags adds tags to the raised Error to further classify
// errors in services such as Sentry or New Relic.
func (self *Error) Tags(tags map[string]any) *Error {
	for key, value := range tags {
		self.tags[key] = fmt.Sprintf("%v", value)
	}

	return self
}

// Is compares whether an error is Error's type.
func (self Error) Is(err error) bool {
	if err == nil {
		return false
	}

	switch other := err.(type) {
	case Error:
		return self.kind == other.kind && self.module == other.module
	case *Error:
		return self.kind == other.kind && self.module == other.module
	}

	return false
}

// Has checks whether an error is wrapped inside the Error itself.
func (self Error) Has(err error) bool {
	if self.Is(err) {
		return true
	}

	if self.cause != nil {
		switch cause := self.cause.(type) {
		case Error:
			return cause.Has(err)
		case *Error:
			return cause.Has(err)
		default:
			return err == cause || err.Error() == cause.Error()
		}
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
			format.Write([]byte(self.StringReport()))
		} else {
			format.Write([]byte(self.StringReport(false)))
		}
	default:
		format.Write([]byte(self.String()))
	}
}

func (self Error) stringReport(all bool, seenTraces map[string]bool) string {
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
			report += cause.stringReport(all, seenTraces)
		case *Error:
			report += cause.stringReport(all, seenTraces)
		default:
			report += "    (Stack trace not available)\n"
			report += "\x1b[0;31m" + cause.Error() + "\x1b[0m (" +
				strings.TrimPrefix(reflect.TypeOf(cause).String(), "*") + ")\n"
		}
	}

	return report
}

// StringReport returns a string containing all the information about the first
// error (including the message, stack trace, extra...) or about all errors
// wrapped within the Error itself (default is all).
func (self Error) StringReport(all ...bool) string {
	_all := true
	if len(all) > 0 {
		_all = all[0]
	}

	seenTraces := make(map[string]bool)

	report := "\x1b[1;91m" + self.String() + "\x1b[0m\n\n"
	report += "Traceback (most recent call last):\n"
	report += self.stringReport(_all, seenTraces)

	return report
}

func (self Error) sentryReport(report *sentry.Event) {
	if self.cause != nil {
		switch cause := self.cause.(type) {
		case Error:
			cause.sentryReport(report)
		case *Error:
			cause.sentryReport(report)
		default:
			report.Exception = append(report.Exception, sentry.Exception{
				Type:  strings.TrimPrefix(reflect.TypeOf(cause).String(), "*"),
				Value: cause.Error(),
			})
		}
	}

	for key, value := range self.extra {
		report.Extra[key] = value
	}

	for key, value := range self.tags {
		report.Tags[key] = value
	}

	var stackTrace *sentry.Stacktrace
	if len(self.stackTrace) > 0 {
		stackTrace = &sentry.Stacktrace{
			Frames: make([]sentry.Frame, 0, len(self.stackTrace)),
		}

		for i := len(self.stackTrace) - 1; i >= 0; i-- {
			stackTrace.Frames = append(stackTrace.Frames, sentry.NewFrame(runtime.Frame{
				Function: self.stackTrace[i].function,
				File:     self.stackTrace[i].file,
				Line:     self.stackTrace[i].line,
			}))
		}
	}

	report.Exception = append(report.Exception, sentry.Exception{
		Type:       self.kind,
		Value:      self.String(),
		Module:     self.module,
		Stacktrace: stackTrace,
	})
}

// SentryReport returns a Sentry Event containing all the information about the
// first error and all errors wrapped within itself (including the types, packages
// messages, stack traces, extra, tags...).
func (self Error) SentryReport() *sentry.Event {
	report := sentry.NewEvent()
	report.Level = sentry.LevelError
	report.Message = _ANSI_COLOR_PATTERN.ReplaceAllString(self.StringReport(), "")
	report.Tags["package"] = self.module
	self.sentryReport(report)

	return report
}
