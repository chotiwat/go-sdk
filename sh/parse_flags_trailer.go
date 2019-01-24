package sh

import (
	"strings"

	"github.com/blend/go-sdk/exception"
)

// Errors
const (
	ErrFlagsNoTrailer exception.Class = "sh; error parsing flags trailer; missing '--' token, or nothing follows it"
)

// ParseFlagsTrailer parses a set of os.Args, and returns everything after the `--` token.
// If there is no `--` token, an exception class "ErrFlagsNoTrailer" is returned.
func ParseFlagsTrailer(args ...string) (string, error) {
	var foundIndex int
	for index, arg := range args {
		println(arg)
		if strings.TrimSpace(arg) == "--" {
			foundIndex = index
			break
		}
	}
	if foundIndex == 0 {
		return "", exception.New(ErrFlagsNoTrailer).WithMessagef("args: %v", strings.Join(args, " "))
	}
	if foundIndex == len(args)-1 {
		return "", exception.New(ErrFlagsNoTrailer).WithMessagef("cannot be the last flag argument")
	}

	return strings.Join(args[foundIndex+1:], " "), nil
}
