package cli

import (
	"fmt"
	"net/url"
	"strconv"
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

// ParseURL uses standard library [url.Parse] function to parse and
// return the *url.URL value represented by the given string.
func ParseURL(s string) (any, error) {
	return url.Parse(s)
}
