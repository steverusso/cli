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
//	--opt       // long form boolean option "opt"
//	-o          // short form boolean option "o"
//	--opt=val   // long form non-boolean option with value of "val"
//	--opt val   // same as above, non-boolean options can provide their value as the next command line argument
//	-a -b       // two short form boolean options, "a" and "b"
//	-ab         // either same as above, or short form non-boolean option "a" with value of "b" (depends on specified command structure)
//
// # Basic Usage
//
//	c := cli.New().
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
//		ParseOrExit()
package cli

import (
	"errors"
	"fmt"
	"iter"
	"os"
	"slices"
	"strings"
)

// A CommandInfo holds information about the schema of a CLI command. This includes usage
// information that can form a help message, as well as which options and arguments or
// subcommands it should expect when parsing. This is data that must be known when parsing
// in order to properly parse command line arguments. For example, consider the following
// argument list: {"-a", "b"}. Is "-a" a boolean option and "b" an argument to the
// command? Or is "b" the option value being provided to the non-boolean option "-a"? The
// only way for the parser to know is to follow an outline that states which one it is.
//
// The methods on CommandInfo are available to guide library consumers through creating
// this command schema. Once built, a CommandInfo would typically be used by calling
// [CommandInfo.ParseOrExit] or [CommandInfo.Parse].
type CommandInfo struct {
	Name      string
	Path      []string
	HelpUsage []string
	HelpBlurb string
	HelpExtra string
	Opts      []InputInfo
	Args      []InputInfo
	Subcmds   []CommandInfo

	// By default, when parsing command line arguments against a CommandInfo that has
	// subcommands defined, an error will be returned if there is no subcommand provided.
	// However, with this field set to true, there will not be a parsing error if a
	// subcommand argument is completely absent, and the parsed subcommand field Subcmd on
	// Command will be nil.
	IsSubcmdOptional bool

	isPrepped bool
}

type InputInfo struct {
	ID         string
	NameShort  byte
	NameLong   string
	HelpBlurb  string
	EnvVar     string
	IsBoolOpt  bool
	IsRequired bool

	StrDefault    string
	HasStrDefault bool

	ValueName   string
	ValueParser ValueParser

	// If an input is encountered during parsing that has either HelpGen or Versioner
	// set, the parser will return HelpOrVersionRequested with whatever either of those
	// functions return as the Msg. See CommandInfo.ParseThese to learn more.
	//
	// Note: adding an input that has HelpGen set to a CommandInfo will prevent this
	// library from automatically adding the DefaultHelpInput to that command.
	HelpGen   HelpGenerator
	Versioner Versioner
}

// ValueParser describes any function that takes a string and returns some value or an
// error. This is the signature of any input value parser. See [ParseBool], [ParseInt],
// and the other provided parsers for some examples.
type ValueParser = func(string) (any, error)

// HelpGenerator describes any function that will return a help message based on the
// [Input] that triggered it and the [CommandInfo] of which it is a member. See
// [DefaultHelpGenerator] for an example.
type HelpGenerator = func(Input, *CommandInfo) string

// Versioner describes any function that will return a version string based on
// the [Input] that triggered it. See [DefaultVersionOpt] for an example.
type Versioner = func(Input) string

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

// Get gets the parsed input value with the given id in the given Command and converts the
// value to the given type T through an untested type assertion. This function will panic
// if the value isn't found or if the value can't be converted to type T. Therefore, this
// function would typically only be used to retrieve parsed values for inputs that are
// required with (which means there will be a parsing error first if there is no value
// present) or that have a default value specified (which means there will always be at
// least one parsed value no matter what). In those cases, this function will be
// completely safe to use.
//
// To check whether the value is found without panicking if it isn't, see [Lookup].
func Get[T any](c *Command, id string) T {
	if v, ok := Lookup[T](c, id); ok {
		return v
	}
	panic("no parsed input value for id '" + id + "'")
}

// GetOr looks for a parsed input value with the given id in the given Command and
// converts the value to the given type T through an untested type assertion (so this
// will panic if the value is found but can't be converted to type T). If the value
// isn't found, the given fallback value will be returned. To check whether the value
// is found instead of using a fallback value, see [Lookup].
func GetOr[T any](c *Command, id string, fallback T) T {
	if v, ok := Lookup[T](c, id); ok {
		return v
	}
	return fallback
}

// GetOrFunc is like [GetOr] in that it looks for a parsed input value with the given id
// in the given Command and converts the value to the given type T through an untested
// type assertion (so this will panic if the value is found but can't be converted to
// type T). However, if the value isn't found, it will run the provided function fn in
// order to create and return the fallback value. To check whether the value is found
// instead of using a fallback, see [Lookup].
func GetOrFunc[T any](c *Command, id string, fn func() T) T {
	if v, ok := Lookup[T](c, id); ok {
		return v
	}
	return fn()
}

// GetAll returns all parsed values present for the given id. It converts each value to
// the given type T through an untested type assertion (so this will panic if any value
// found can't be converted to type T).
func GetAll[T any](c *Command, id string) []T {
	vals := make([]T, 0, len(c.Inputs)/3)
	for i := range c.Inputs {
		if c.Inputs[i].ID == id {
			vals = append(vals, c.Inputs[i].Value.(T))
		}
	}
	return vals
}

// GetAllSeq returns an iterator over each parsed value that has the given id. The
// iterator yields the same values that would be returned by [GetAll](c, id) but without
// constructing the slice.
func GetAllSeq[T any](c *Command, id string) iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := range c.Inputs {
			if c.Inputs[i].ID == id {
				if !yield(c.Inputs[i].Value.(T)) {
					return
				}
			}
		}
	}
}

// Fatal logs the given value to Stderr prefixed by "error: "
// and then exits the program with the given code.
func Fatal(code int, v any) {
	fmt.Fprintf(os.Stderr, "error: %v\n", v)
	os.Exit(code)
}

// ParseOrExit calls [CommandInfo.ParseTheseOrExit] using os.Args as the command
// line arguments. See that method's documentation for more info.
func (in CommandInfo) ParseOrExit() *Command {
	return in.ParseTheseOrExit(os.Args[1:]...)
}

// ParseTheseOrExit parses input against this CommandInfo using args as the command line
// arguments. If there is a [HelpOrVersionRequested] error, it will print the message and
// exit with status code 0. If there was any other error, it will print the error's
// message to Stderr and exit with status code 1.
func (in CommandInfo) ParseTheseOrExit(args ...string) *Command {
	c, err := in.ParseThese(args...)
	if err != nil {
		if e, ok := err.(HelpOrVersionRequested); ok {
			fmt.Print(e.Msg)
			os.Exit(0)
		} else {
			Fatal(1, err)
		}
	}
	return c
}

// Parse calls [CommandInfo.ParseThese] using os.Args as the command
// line arguments. See that method's documentation for more info.
func (in *CommandInfo) Parse() (*Command, error) {
	return in.ParseThese(os.Args[1:]...)
}

// ParseThese prepares and validates this CommandInfo if it hasn't been already. This
// involves setting the Path field of this command and all subcommands, as well as
// ensuring there are no logical errors in the structure of this command (such as a
// command containing duplicate option names). This method will panic if there are any
// schema errors in this CommandInfo.
//
// Assuming a clean schema, this method then parses input against this CommandInfo using
// args as the command line arguments. If there is a help or version input found on any
// command level, this function will return a [HelpOrVersionRequested] error.
func (in *CommandInfo) ParseThese(args ...string) (*Command, error) {
	if !in.isPrepped {
		in.prepareAndValidate()
		in.isPrepped = true
	}
	c := &Command{
		Inputs: make([]Input, 0, len(args)),
	}
	err := parse(in, c, args)
	return c, err
}

// HelpOrVersionRequested is returned by the parsing code
// to signal that a help or version option was encountered.
type HelpOrVersionRequested struct {
	Msg string
}

func (h HelpOrVersionRequested) Error() string {
	return h.Msg
}

func lookupOptionByShortName(in *CommandInfo, shortName byte) *InputInfo {
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
				optName := arg[charIdx]
				optInfo := lookupOptionByShortName(c, optName)
				if optInfo == nil {
					return UnknownOptionError{CmdInfo: c, Name: "-" + string(arg[charIdx])}
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
							return MissingOptionValueError{CmdInfo: c, Name: string(optName)}
						}
					} else {
						rawValue = arg[charIdx+1:]
						skipRest = true
					}
				}

				pi, err := newInput(optInfo, ParsedFrom{Opt: string(optName)}, rawValue)
				if err != nil {
					return fmt.Errorf("parsing option '%c': %w", optName, err)
				}

				if optInfo.HelpGen != nil {
					return HelpOrVersionRequested{
						Msg: optInfo.HelpGen(pi, c),
					}
				}
				if optInfo.Versioner != nil {
					return HelpOrVersionRequested{
						Msg: optInfo.Versioner(pi),
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
		if len(name) == 1 {
			optInfo = lookupOptionByShortName(c, name[0])
		} else {
			for i := range c.Opts {
				if name == c.Opts[i].NameLong {
					optInfo = &c.Opts[i]
					break
				}
			}
		}
		if optInfo == nil {
			return UnknownOptionError{CmdInfo: c, Name: args[i]}
		}

		var rawValue string
		if eqIdx != -1 {
			rawValue = arg[eqIdx+1:]
		} else if !optInfo.IsBoolOpt {
			i++
			if i < len(args) {
				rawValue = args[i]
			} else {
				return MissingOptionValueError{CmdInfo: c, Name: name}
			}
		}

		pi, err := newInput(optInfo, ParsedFrom{Opt: name}, rawValue)
		if err != nil {
			return fmt.Errorf("parsing option '%s': %w", name, err)
		}

		if optInfo.HelpGen != nil {
			return HelpOrVersionRequested{
				Msg: optInfo.HelpGen(pi, c),
			}
		}
		if optInfo.Versioner != nil {
			return HelpOrVersionRequested{
				Msg: optInfo.Versioner(pi),
			}
		}

		p.Inputs = append(p.Inputs, pi)
	}

	// check that all required options were provided
	var missing []string
	for i := range c.Opts {
		if c.Opts[i].IsRequired {
			if !hasOpt(p, c.Opts[i].ID) {
				var name string
				if c.Opts[i].NameLong != "" {
					name = "--" + c.Opts[i].NameLong
				} else {
					name = "-" + string(c.Opts[i].NameShort)
				}
				missing = append(missing, name)
			}
		}
	}
	var errMissingOpts error
	if len(missing) > 0 {
		errMissingOpts = MissingOptionsError{CmdInfo: c, Names: missing}

		// If we are about to parse positional arguments instead of subcommands,
		// we can just return this error right now. Otherwise we have to wait
		// to see if a subcommand requests help.
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
					return MissingArgsError{CmdInfo: c, Names: missing}
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
		if c.IsSubcmdOptional {
			return nil
		}
		return ErrNoSubcmd
	}

	var subcmdInfo *CommandInfo
	for i := range c.Subcmds {
		if c.Subcmds[i].Name == rest[0] {
			subcmdInfo = &c.Subcmds[i]
			break
		}
	}
	if subcmdInfo == nil {
		return UnknownSubcmdError{CmdInfo: c, Name: rest[0]}
	}
	p.Subcmd = &Command{
		Inputs: make([]Input, 0, len(rest)),
		Name:   rest[0],
	}

	// If we have an error from parsing this command (from above), only return it so long
	// as no subcommand has requested a help message.
	errFromSubcmd := parse(subcmdInfo, p.Subcmd, rest[1:])
	if errMissingOpts != nil {
		if _, ok := errFromSubcmd.(HelpOrVersionRequested); !ok {
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

type UnknownSubcmdError struct {
	CmdInfo *CommandInfo
	Name    string
}

func (usce UnknownSubcmdError) Error() string {
	return strings.Join(usce.CmdInfo.Path, " ") + ": unknown subcommand '" + usce.Name + "'"
}

type UnknownOptionError struct {
	CmdInfo *CommandInfo
	Name    string
}

func (uoe UnknownOptionError) Error() string {
	return strings.Join(uoe.CmdInfo.Path, " ") + ": unknown option '" + uoe.Name + "'"
}

type MissingOptionValueError struct {
	CmdInfo *CommandInfo
	Name    string
}

func (mov MissingOptionValueError) Error() string {
	return strings.Join(mov.CmdInfo.Path, " ") + ": option '" + mov.Name + "' requires a value"
}

type MissingOptionsError struct {
	CmdInfo *CommandInfo
	Names   []string
}

func (moe MissingOptionsError) Error() string {
	return fmt.Sprintf("%s: missing the following required options: %s",
		strings.Join(moe.CmdInfo.Path, " "), strings.Join(moe.Names, ", "))
}

func (moe MissingOptionsError) Is(err error) bool {
	if e, ok := err.(MissingOptionsError); ok {
		return moe.CmdInfo == e.CmdInfo && slices.Equal(moe.Names, e.Names)
	}
	return false
}

type MissingArgsError struct {
	CmdInfo *CommandInfo
	Names   []string
}

func (mae MissingArgsError) Error() string {
	return fmt.Sprintf("%s: missing the following required arguments: %s",
		strings.Join(mae.CmdInfo.Path, " "), strings.Join(mae.Names, ", "))
}

func (mae MissingArgsError) Is(err error) bool {
	if e, ok := err.(MissingArgsError); ok {
		return mae.CmdInfo == e.CmdInfo && slices.Equal(mae.Names, e.Names)
	}
	return false
}
