package logger

import (
	"fmt"
)

const (
	gray   = "\033[38;2;134;134;134m"
	red    = "\033[38;2;255;85;85m"
	green  = "\033[38;2;80;250;123m"
	yellow = "\033[38;2;235;208;9m"
	blue   = "\033[38;2;139;233;253m"
	reset  = "\033[0m"
)

// Error prints a formatted error message.
func Error(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s[Error]%s %s%s%s\n", gray, reset, red, msg, reset)
}

// Success prints a formatted success message.
func Success(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s[Success]%s %s%s%s\n", gray, reset, green, msg, reset)
}

// Warning prints a formatted warning message.
func Warning(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s[Warning]%s %s%s%s\n", gray, reset, yellow, msg, reset)
}

// Info prints a formatted info message.
func Info(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s%s%s\n", blue, msg, reset)
}
