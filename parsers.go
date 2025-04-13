package cli

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

// ParseBool returns the boolean value represented by the given string.
// See [strconv.ParseBool] for more info.
func ParseBool(s string) (any, error) {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return false, fmt.Errorf(`invalid boolean value "%s"`, s)
	}
	return b, nil
}

// ParseFloat32 returns the float32 value represented by the given string.
// It unwraps any [strconv.NumError] returned by [strconv.ParseFloat] for
// a slightly cleaner error message.
func ParseFloat32(s string) (any, error) {
	f64, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return 0, numError(err)
	}
	return float32(f64), nil
}

// ParseFloat64 returns the float64 value represented by the given string.
// It unwraps any [strconv.NumError] returned by [strconv.ParseFloat] for
// a slightly cleaner error message.
func ParseFloat64(s string) (any, error) {
	f64, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, numError(err)
	}
	return f64, nil
}

// ParseInt returns the int value represented by the given string.
// It unwraps any [strconv.NumError] returned by [strconv.ParseInt] for
// a slightly cleaner error message.
func ParseInt(s string) (any, error) {
	i64, err := strconv.ParseInt(s, 0, 0)
	if err != nil {
		return 0, numError(err)
	}
	return int(i64), nil
}

// ParseUint returns the uint value represented by the given string.
// It unwraps any [strconv.NumError] returned by [strconv.ParseUint] for
// a slightly cleaner error message.
func ParseUint(s string) (any, error) {
	u64, err := strconv.ParseUint(s, 0, 0)
	if err != nil {
		return 0, numError(err)
	}
	return uint(u64), nil
}

func numError(err error) error {
	if ne, ok := err.(*strconv.NumError); ok {
		return ne.Err
	}
	return err
}

// ParseDuration uses the standard library [time.ParseDuration] function to
// parse and return the time.Duration value represented by the given string.
func ParseDuration(s string) (any, error) {
	return time.ParseDuration(s)
}

// NewTimeParser returns a [ValueParser] that will use the standard library
// [time.Parse] function with the given layout string to parse and return a
// time.Time from a given string.
func NewTimeParser(layout string) ValueParser {
	return func(s string) (any, error) {
		return time.Parse(layout, s)
	}
}

// ParseURL uses the standard library [url.Parse] function to parse
// and return the *url.URL value represented by the given string.
func ParseURL(s string) (any, error) {
	return url.Parse(s)
}

// NewFileParser returns a [ValueParser] that will treat an input string as a
// file path and use given parser to parse the content of the file at that path.
func NewFileParser(vp ValueParser) ValueParser {
	return func(fileName string) (any, error) {
		b, err := os.ReadFile(fileName)
		if err != nil {
			return nil, err
		}
		s := string(b)
		// Drop trailing newline.
		if len(s) > 0 && s[len(s)-1] == '\n' {
			s = s[:len(s)-1]
		}
		// If we don't have a value parser, just return the raw string.
		if vp == nil {
			return s, nil
		}
		return vp(s)
	}
}
