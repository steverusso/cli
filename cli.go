// # Input Sources and Precedence
//
// This library will always parse a program's command line arguments for Inputs. However,
// inputs can additionally be parsed from environment variables or default values, in that
// order of precedence. For example, if an input can be parsed from all of those places
// (command line argument, environment variable, and default value), all will be parsed,
// but the value from the command line will take precedence over the value from the
// environment variable, and the value from the environment variable will take precedence
// over the default value.
//
// # Command Line Syntax
//
// Command line arguments are classified as one of the following:
//  1. Options: arguments that begin with "-" or "--" and may or may not require a value.
//  2. Positional Arguments: arguments that are identified by the order in which they
//     appear among other positional arguments.
//  3. Subcommands: All arguments that follow belong to this command.
//
// Command line arguments are parsed as options until a positional argument, subcommand,
// or an argument of just "--" is encountered. In other words, any options that belong to
// a command must come before any of that command's positional arguments or subcommands.
//
// Positional arguments and subcommands are mutually exclusive since allowing both to
// exist at once would invite unnecessary ambiguity when parsing because there's no
// reliable way to tell if an argument would be a positional argument or the name of a
// subcommand. Furthermore, positional arguments that are required must appear before any
// optional ones since it would be impossible to tell when parsing whether a positional
// argument is filling the spot of a required one or an optional one. Therefore, the
// format of a command is structured like this:
//
//	command [options] [<required_pos_args> [optional_pos_args] [any_surplus_post_args...] | subcommand ...]
//
// # Options
//
// There are only two types of options in terms of syntax:
//  1. boolean: Rather than providing some value, the mere presence of this type of option
//     means something. For example, the "--all" in "ls --all" does not take a value; it
//     just modifies the list command to list "all" files.
//  2. non-boolean: This type of option requires a value. For example, in "ls
//     --hide go.sum", the option "--hide" requires a file name or pattern.
//
// Non-boolean options must have a value attached. In other words, while options
// themselves can either be required or optional, there is no such thing as an option that
// may or may not have a value.
//
// Options can have appear in one of two forms and can have a name for each form: long or
// short. Typically an option's long name is more than one character, but an option's
// short name can only be a single character. Long form options are provided by prefixing
// the option's name with two hyphens ("--"), and they must appear one at a time,
// separately. Short form options are provided by prefixing the options short name with a
// single hyphen ("-"), and they can be "stacked", meaning under certain conditions, they
// can appear one after the other in the same command line argument.
//
// The following are some common ways of how options can be provided:
//
//	--opt      // long form boolean option
//	--opt=v    // long form non-boolean option with value of "v"
//	--opt v    // same as above, non-boolean options can provide their value as the next command line argument
//	-a         // short form boolean option "a"
//	-a -b      // two short form boolean option "a" and "b"
//	-ab        // either same as above, or short form non-boolean option "a" with value of "b" (depends on specified command structure)
//
// # Basic Usage
//
//	p := cli.NewCmd("full").
//		Help("A full example program").
//		Opt(cli.NewBoolOpt("yes").
//			Short('y').
//			Help("A boolean option on the root command.").
//			Env("YES")).
//		Opt(cli.NewOpt("str").
//			Short('s').
//			Help("A string option on the root command.").
//			Env("STR")).
//		Subcmd(cli.NewCmd("nodat").
//			Help("Subcommand with no data")).
//		Subcmd(cli.NewCmd("floats").
//			Help("Print values for each supported float type (both options required).").
//			Opt(cli.NewFloat32Opt("f32").Help("32 bit float").Required()).
//			Opt(cli.NewFloat64Opt("f64").Help("64 bit float").Required())).
//		Subcmd(cli.NewCmd("ints").
//			Help("Print values for each supported signed integer type.").
//			Opt(cli.NewIntOpt("int").Help("Print the given int value."))).
//		Build().
//		ParseOrExit()
package cli

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
)

type CommandInfo struct {
	Name      string
	Path      []string
	HelpUsage []string
	HelpBlurb string
	HelpExtra string
	Opts      []InputInfo
	Args      []InputInfo
	Subcmds   []CommandInfo

	isPrepped bool
}

type InputInfo struct {
	ID         string
	NameShort  string
	NameLong   string
	HelpBlurb  string
	EnvVar     string
	IsBoolOpt  bool
	IsRequired bool

	StrDefault    string
	HasStrDefault bool

	ValueName   string
	ValueParser ValueParser

	HelpGen HelpGenerator
}

// ValueParser describes any function that takes a string and returns some type or an
// error. This is the signature of any input value parser. See [ParseBool], [ParseInt],
// and the other provided parsers for some examples.
type ValueParser = func(string) (any, error)

// HelpGenerator describes any function that will return a help message based on the
// [Input] that triggered it and the [CommandInfo] of which it is a member. See
// [DefaultHelpGenerator] for an example.
type HelpGenerator = func(Input, *CommandInfo) string

// Command is a parsed command structure.
type Command struct {
	Name    string
	Inputs  []Input
	Surplus []string
	Subcmd  *Command
}

// Input is a parsed option value or positional argument value along with other
// information such as the input ID it corresponds to and where it was parsed from.
type Input struct {
	Value    any
	ID       string
	RawValue string
	From     ParsedFrom
}

// ParsedFrom describes where an Input is parsed from. The place it came from will be the
// only non-zero field of this struct.
type ParsedFrom struct {
	Env     string // Came from this env var's name.
	Opt     string // Came from this provided option name.
	Arg     int    // Appeared as the nth positional argument starting from 1.
	Default bool   // Came from a provided default value.
}

// Lookup looks for a parsed input value with the given id in the given Command and
// converts the value to the given type T through an untested type assertion (so this
// will panic if the value is found and can't be converted to type T). So if the input
// is present, the typed value will be returned and the boolean will be true. Otherwise,
// the zero value of type T will be returned and the boolean will be false.
func Lookup[T any](c *Command, id string) (T, bool) {
	for i := len(c.Inputs) - 1; i >= 0; i-- {
		if c.Inputs[i].ID == id {
			return c.Inputs[i].Value.(T), true
		}
	}
	var zero T
	return zero, false
}

// Get gets the parsed input value with the given id in the given Command and converts
// the value to the given type T through an untested type assertion. This function will
// panic if the value isn't found or if the value can't be converted to type T.
// To check whether the value is found instead of panicking, see [Lookup].
func Get[T any](c *Command, id string) T {
	if v, ok := Lookup[T](c, id); ok {
		return v
	}
	panic("no parsed input value for id '" + id + "'")
}

// GetOr looks for a parsed input value with the given id in the given Command and
// converts the value to the given type T through an untested type assertion (so this
// will panic if the value is found and can't be converted to type T). If the value
// isn't found, the given fallback value will be returned. To check whether the value
// is found instead of using a fallback value, see [Lookup].
func GetOr[T any](c *Command, id string, fallback T) T {
	if v, ok := Lookup[T](c, id); ok {
		return v
	}
	return fallback
}

// ParseOrExit will parse input based on this CommandInfo. If help was requested, it
// will print the help message and exit the program successfully (status code 0). If
// there is any other error, it will print the error and exit the program with failure
// (status code 1). The input parameter semantics are the same as [CommandInfo.Parse].
func (in CommandInfo) ParseOrExit(args ...string) *Command {
	c, err := in.Parse(args...)
	if err != nil {
		if e, ok := err.(HelpRequestError); ok {
			fmt.Print(e.HelpMsg)
			os.Exit(0)
		} else {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
	return c
}

// ParseOrExit will parse input based on this CommandInfo. If no function arguments
// are provided, the [os.Args] will be used.
func (in *CommandInfo) Parse(args ...string) (*Command, error) {
	if !in.isPrepped {
		in.prepareAndValidate()
		in.isPrepped = true
	}
	if args == nil {
		args = os.Args[1:]
	}
	c := &Command{
		Inputs: make([]Input, 0, len(args)),
	}
	err := parse(in, c, args)
	return c, err
}

type HelpRequestError struct {
	HelpMsg string
}

func (h HelpRequestError) Error() string {
	return h.HelpMsg
}

func lookupOptionByShortName(in *CommandInfo, shortName string) *InputInfo {
	for i := range in.Opts {
		if in.Opts[i].NameShort == shortName {
			return &in.Opts[i]
		}
	}
	return nil
}

func hasOpt(c *Command, id string) bool {
	for i := range c.Inputs {
		if c.Inputs[i].ID == id {
			return true
		}
	}
	return false
}

func hasArg(c *Command, id string) bool {
	for i := range c.Inputs {
		if c.Inputs[i].ID == id {
			return true
		}
	}
	return false
}

func parse(c *CommandInfo, p *Command, args []string) error {
	// set any defaults
	for i := range c.Opts {
		if c.Opts[i].HasStrDefault {
			dv := c.Opts[i].StrDefault
			pi, err := newInput(&c.Opts[i], ParsedFrom{Default: true}, dv)
			if err != nil {
				return fmt.Errorf("parsing default value '%s' for option '%s': %w", dv, c.Opts[i].ID, err)
			}
			p.Inputs = append(p.Inputs, pi)
		}
	}
	for i := range c.Args {
		if c.Args[i].HasStrDefault {
			dv := c.Args[i].StrDefault
			pi, err := newInput(&c.Args[i], ParsedFrom{Default: true}, dv)
			if err != nil {
				return fmt.Errorf("parsing default value '%s' for arg '%s': %w", dv, c.Args[i].ID, err)
			}
			p.Inputs = append(p.Inputs, pi)
		}
	}

	// grab any envs
	for i := range c.Opts {
		if c.Opts[i].EnvVar != "" {
			if v, ok := os.LookupEnv(c.Opts[i].EnvVar); ok {
				pi, err := newInput(&c.Opts[i], ParsedFrom{Env: c.Opts[i].EnvVar}, v)
				if err != nil {
					return fmt.Errorf("using env var '%s': %w", c.Opts[i].EnvVar, err)
				}
				p.Inputs = append(p.Inputs, pi)
			}
		}
	}
	for i := range c.Args {
		if c.Args[i].EnvVar != "" {
			if v, ok := os.LookupEnv(c.Args[i].EnvVar); ok {
				pi, err := newInput(&c.Args[i], ParsedFrom{Env: c.Args[i].EnvVar}, v)
				if err != nil {
					return fmt.Errorf("using env var '%s': %w", c.Args[i].EnvVar, err)
				}
				p.Inputs = append(p.Inputs, pi)
			}
		}
	}

	// parse options
	var i int
	for ; i < len(args); i++ {
		arg := args[i]
		if len(arg) == 0 || arg[0] != '-' {
			break
		}

		// Drop the first '-' char. If that's all there was, it is treated as a positional
		// argument or subcommand, and we stop parsing.
		arg = arg[1:]
		if len(arg) == 0 {
			break
		}

		// If this option begins with only one hyphen, and if there is more after the
		// first letter that isn't a '=' to set the value of a short option, then we are
		// going to parse this command line argument as either a bunch of stacked short
		// options (e.g. `-abc` instead of `-a -b -c`) or a short option with the value
		// attached (e.g. `-avalue` instead of `-a value` or `-a=value`).
		//
		// In other words, make sure this is something like `-ab` and not `-a` or `-a=b`
		// which would be handled by the rest of the option parsing code in this loop.
		if arg[0] != '-' && len(arg) > 1 && arg[1] != '=' {
			for charIdx := range arg {
				optName := string(arg[charIdx])
				optInfo := lookupOptionByShortName(c, optName)
				if optInfo == nil {
					return UnknownOptionError{Name: "-" + string(arg[charIdx])}
				}

				// If this is another bool option, the raw value will be empty. If this is
				// a non-bool option, we are going to take the rest of this argument as
				// the raw value, and in that case skip processing the rest of the chars
				// as option names. If there is nothing left and this is the last
				// character of the argument, then we'll take the next argument.
				var rawValue string
				var skipRest bool
				if !optInfo.IsBoolOpt {
					if charIdx == len(arg)-1 {
						i++
						if i < len(args) {
							rawValue = args[i]
						} else {
							return MissingOptionValueError{Name: optName}
						}
					} else {
						rawValue = arg[charIdx+1:]
						skipRest = true
					}
				}

				pi, err := newInput(optInfo, ParsedFrom{Opt: optName}, rawValue)
				if err != nil {
					return fmt.Errorf("parsing option '%s': %w", optName, err)
				}

				if optInfo.HelpGen != nil {
					return HelpRequestError{
						HelpMsg: optInfo.HelpGen(pi, c),
					}
				}

				p.Inputs = append(p.Inputs, pi)

				if skipRest {
					break
				}
			}

			continue
		}

		if len(arg) > 0 && arg[0] == '-' {
			arg = arg[1:] // drop the second '-'
			if len(arg) == 0 {
				i++
				break // '--' means stop parsing options
			}
		}

		eqIdx := -1
		for z := range arg {
			if arg[z] == '=' {
				eqIdx = z
				break
			}
		}
		name := arg
		if eqIdx != -1 {
			name = arg[:eqIdx]
		}

		var optInfo *InputInfo
		for i := range c.Opts {
			if name == c.Opts[i].NameShort || name == c.Opts[i].NameLong {
				optInfo = &c.Opts[i]
				break
			}
		}
		if optInfo == nil {
			return UnknownOptionError{Name: args[i]}
		}

		var rawValue string
		if eqIdx != -1 {
			rawValue = arg[eqIdx+1:]
		} else if !optInfo.IsBoolOpt {
			i++
			if i < len(args) {
				rawValue = args[i]
			} else {
				return MissingOptionValueError{Name: name}
			}
		}

		pi, err := newInput(optInfo, ParsedFrom{Opt: name}, rawValue)
		if err != nil {
			return fmt.Errorf("parsing option '%s': %w", name, err)
		}

		if optInfo.HelpGen != nil {
			return HelpRequestError{
				HelpMsg: optInfo.HelpGen(pi, c),
			}
		}

		p.Inputs = append(p.Inputs, pi)
	}

	var errMissingOpts error

	// check that all required options were provided
	var missing []string
	for i := range c.Opts {
		if c.Opts[i].IsRequired {
			if !hasOpt(p, c.Opts[i].ID) {
				var name string
				if c.Opts[i].NameLong != "" {
					name = "--" + c.Opts[i].NameLong
				} else {
					name = "-" + c.Opts[i].NameShort
				}
				missing = append(missing, name)
			}
		}
	}
	if len(missing) > 0 {
		errMissingOpts = MissingOptionsError{Names: missing}

		// If we are parsing positional arguments instead of subcommands, we can just
		// return this error right now. Otherwise we have to wait to see if a subcommand
		// requests help.
		if len(c.Subcmds) == 0 {
			return errMissingOpts
		}
	}

	rest := args[i:]
	if len(c.Subcmds) == 0 {
		for i = 0; i < len(c.Args); i++ {
			if i < len(rest) {
				rawArg := rest[i]
				pi, err := newInput(&c.Args[i], ParsedFrom{Arg: i + 1}, rawArg)
				if err != nil {
					return fmt.Errorf("parsing positional argument #%d '%s': %w", i+1, rawArg, err)
				}
				p.Inputs = append(p.Inputs, pi)
			} else if c.Args[i].IsRequired {
				var missing []string
				for ; i < len(c.Args); i++ {
					if !c.Args[i].IsRequired {
						break
					}
					if !hasArg(p, c.Args[i].ID) {
						missing = append(missing, c.Args[i].ValueName)
					}
				}
				if len(missing) > 0 {
					return MissingArgsError{Names: missing}
				}
				return nil
			} else {
				break
			}
		}
		if len(rest) > i {
			p.Surplus = rest[i:]
		}
		return nil
	}

	if len(rest) < 1 {
		return ErrNoSubcmd
	}
	p.Subcmd = &Command{
		Inputs: make([]Input, 0, len(rest)),
		Name:   rest[0],
	}

	var subcmdInfo *CommandInfo
	for i := range c.Subcmds {
		if c.Subcmds[i].Name == p.Subcmd.Name {
			subcmdInfo = &c.Subcmds[i]
			break
		}
	}
	if subcmdInfo == nil {
		return UnknownSubcmdError{Name: p.Subcmd.Name}
	}

	// If we have an error from parsing this command (from above), only return it so long
	// as no subcommand has requested a help message.
	errFromSubcmd := parse(subcmdInfo, p.Subcmd, rest[1:])
	if errMissingOpts != nil {
		if _, ok := errFromSubcmd.(HelpRequestError); !ok {
			return errMissingOpts
		}
	}

	return errFromSubcmd
}

func newInput(info *InputInfo, src ParsedFrom, rawValue string) (Input, error) {
	var val any
	var err error

	switch {
	// If we have a value parser, use that.
	case info.ValueParser != nil:
		val, err = info.ValueParser(rawValue)
	// If we don't have a value parser but we know it's a boolean option, use the
	// default boolean parser.
	case info.IsBoolOpt:
		if rawValue != "" {
			val, err = ParseBool(rawValue)
		} else {
			val = true
		}
	// No parser, not a bool option, so we just use the raw string.
	default:
		val = rawValue
	}

	if err != nil {
		return Input{}, err
	}

	return Input{
		ID:       info.ID,
		From:     src,
		RawValue: rawValue,
		Value:    val,
	}, nil
}

var ErrNoSubcmd = errors.New("missing subcommand")

type UnknownSubcmdError struct{ Name string }

func (usce UnknownSubcmdError) Error() string {
	return "unknown subcommand '" + usce.Name + "'"
}

type UnknownOptionError struct{ Name string }

func (uoe UnknownOptionError) Error() string {
	return "unknown option '" + uoe.Name + "'"
}

type MissingOptionValueError struct{ Name string }

func (mov MissingOptionValueError) Error() string {
	return "option '" + mov.Name + "' requires a value"
}

type MissingOptionsError struct{ Names []string }

func (moe MissingOptionsError) Error() string {
	return fmt.Sprintf("missing the following required options: %s", strings.Join(moe.Names, ", "))
}

func (moe MissingOptionsError) Is(err error) bool {
	if e, ok := err.(MissingOptionsError); ok {
		return slices.Equal(moe.Names, e.Names)
	}
	return false
}

type MissingArgsError struct{ Names []string }

func (mae MissingArgsError) Error() string {
	return fmt.Sprintf("missing the following required arguments: %s", strings.Join(mae.Names, ", "))
}

func (mae MissingArgsError) Is(err error) bool {
	if e, ok := err.(MissingArgsError); ok {
		return slices.Equal(mae.Names, e.Names)
	}
	return false
}
