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

type Command struct {
	name      string
	path      []string
	helpUsage []string
	helpBlurb string
	helpExtra string
	opts      []Input
	args      []Input
	subcmds   []Command
}

type Input struct {
	id         string
	nameShort  string
	nameLong   string
	helpBlurb  string
	env        string
	isBoolOpt  bool
	isRequired bool

	rawDefaultValue string
	hasDefaultValue bool

	valueName   string
	valueParser ValueParser

	helpGen HelpGenerator
}

type ValueParser = func(string) (any, error)

type ParsedCommand struct {
	Name    string
	Opts    []ParsedInput
	Args    []ParsedInput
	Surplus []string
	Subcmd  *ParsedCommand
}

type ParsedInput struct {
	Value    any
	ID       string
	RawValue string
	From     ParsedFrom
}

type ParsedFrom struct {
	Env        string // the env var's name
	Opt        string // the provided option name
	Arg        int    // the position starting from 1
	RawDefault bool
}

func (pc *ParsedCommand) Opt(id string) any {
	if v, ok := pc.LookupOpt(id); ok {
		return v
	}
	panic("no parsed option value found for id '" + id + "'")
}

func (pc *ParsedCommand) LookupOpt(id string) (any, bool) {
	for i := len(pc.Opts) - 1; i >= 0; i-- {
		if pc.Opts[i].ID == id {
			return pc.Opts[i].Value, true
		}
	}
	return nil, false
}

func (pc *ParsedCommand) Arg(id string) any {
	if v, ok := pc.LookupArg(id); ok {
		return v
	}
	panic("no parsed argument value found for id '" + id + "'")
}

func (pc *ParsedCommand) LookupArg(id string) (any, bool) {
	for i := len(pc.Args) - 1; i >= 0; i-- {
		if pc.Args[i].ID == id {
			return pc.Args[i].Value, true
		}
	}
	return nil, false
}

type RootCommand struct{ c Command }

// ParseOrExit will parse input based on this RootCommand. If help was requested, it will
// print the help message and exit the program successfully (status code 0). If there is
// any other error, it will print the error and exit the program with failure (status code
// 1). The input parameter semantics are the same as [RootCommand.Parse].
func (c *RootCommand) ParseOrExit(args ...string) ParsedCommand {
	p, err := c.Parse(args...)
	if err != nil {
		if e, ok := err.(HelpRequestError); ok {
			fmt.Print(e.HelpMsg)
			os.Exit(0)
		} else {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
	return p
}

// ParseOrExit will parse input based on this RootCommand. If no function arguments are
// provided, the [os.Args] will be used.
func (c *RootCommand) Parse(args ...string) (ParsedCommand, error) {
	if args == nil {
		args = os.Args[1:]
	}
	p := ParsedCommand{
		Opts: make([]ParsedInput, 0, len(args)),
	}
	err := parse(&c.c, &p, args)
	return p, err
}

type HelpRequestError struct {
	RootCause error
	HelpMsg   string
}

func (h HelpRequestError) Error() string {
	if h.RootCause != nil {
		return h.RootCause.Error()
	}
	return h.HelpMsg
}

func lookupOptionByShortName(c *Command, shortName string) *Input {
	for i := range c.opts {
		if c.opts[i].nameShort == shortName {
			return &c.opts[i]
		}
	}
	return nil
}

func parse(c *Command, p *ParsedCommand, args []string) error {
	// set any defaults
	for i := range c.opts {
		if c.opts[i].hasDefaultValue {
			dv := c.opts[i].rawDefaultValue
			pi, err := newParsedInput(&c.opts[i], ParsedFrom{RawDefault: true}, dv)
			if err != nil {
				return fmt.Errorf("parsing default value '%s' for option '%s': %w", dv, c.opts[i].id, err)
			}
			p.Opts = append(p.Opts, pi)
		}
	}
	for i := range c.args {
		if c.args[i].hasDefaultValue {
			dv := c.args[i].rawDefaultValue
			pi, err := newParsedInput(&c.args[i], ParsedFrom{RawDefault: true}, dv)
			if err != nil {
				return fmt.Errorf("parsing default value '%s' for arg '%s': %w", dv, c.args[i].id, err)
			}
			p.Args = append(p.Args, pi)
		}
	}

	// grab any envs
	for i := range c.opts {
		if c.opts[i].env != "" {
			if v, ok := os.LookupEnv(c.opts[i].env); ok {
				pi, err := newParsedInput(&c.opts[i], ParsedFrom{Env: c.opts[i].env}, v)
				if err != nil {
					return fmt.Errorf("using env var '%s': %w", c.opts[i].env, err)
				}
				p.Opts = append(p.Opts, pi)
			}
		}
	}
	for i := range c.args {
		if c.args[i].env != "" {
			if v, ok := os.LookupEnv(c.args[i].env); ok {
				pi, err := newParsedInput(&c.args[i], ParsedFrom{Env: c.args[i].env}, v)
				if err != nil {
					return fmt.Errorf("using env var '%s': %w", c.args[i].env, err)
				}
				p.Args = append(p.Args, pi)
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
				if !optInfo.isBoolOpt {
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

				pi, err := newParsedInput(optInfo, ParsedFrom{Opt: optName}, rawValue)
				if err != nil {
					return fmt.Errorf("parsing option '%s': %w", optName, err)
				}

				if optInfo.helpGen != nil {
					return HelpRequestError{
						HelpMsg: optInfo.helpGen(pi, c),
					}
				}

				p.Opts = append(p.Opts, pi)

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

		var optInfo *Input
		for i := range c.opts {
			if name == c.opts[i].nameShort || name == c.opts[i].nameLong {
				optInfo = &c.opts[i]
				break
			}
		}
		if optInfo == nil {
			return UnknownOptionError{Name: args[i]}
		}

		var rawValue string
		if eqIdx != -1 {
			rawValue = arg[eqIdx+1:]
		} else if !optInfo.isBoolOpt {
			i++
			if i < len(args) {
				rawValue = args[i]
			} else {
				return MissingOptionValueError{Name: name}
			}
		}

		pi, err := newParsedInput(optInfo, ParsedFrom{Opt: name}, rawValue)
		if err != nil {
			return fmt.Errorf("parsing option '%s': %w", name, err)
		}

		if optInfo.helpGen != nil {
			return HelpRequestError{
				HelpMsg: optInfo.helpGen(pi, c),
			}
		}

		p.Opts = append(p.Opts, pi)
	}

	var errMissingOpts error

	// check that all required options were provided
	var missing []string
	for i := range c.opts {
		if c.opts[i].isRequired {
			_, ok := p.LookupOpt(c.opts[i].id)
			if !ok {
				var name string
				if c.opts[i].nameLong != "" {
					name = "--" + c.opts[i].nameLong
				} else {
					name = "-" + c.opts[i].nameShort
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
		if len(c.subcmds) == 0 {
			return errMissingOpts
		}
	}

	rest := args[i:]
	if len(c.subcmds) == 0 {
		for i = 0; i < len(c.args); i++ {
			if i < len(rest) {
				rawArg := rest[i]
				pi, err := newParsedInput(&c.args[i], ParsedFrom{Arg: i + 1}, rawArg)
				if err != nil {
					return fmt.Errorf("parsing positional argument #%d '%s': %w", i+1, rawArg, err)
				}
				p.Args = append(p.Args, pi)
			} else if c.args[i].isRequired {
				var missing []string
				for ; i < len(c.args); i++ {
					if !c.args[i].isRequired {
						break
					}
					_, ok := p.LookupArg(c.args[i].id)
					if !ok {
						missing = append(missing, c.args[i].valueName)
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
	p.Subcmd = &ParsedCommand{
		Opts: make([]ParsedInput, 0, len(rest)),
		Name: rest[0],
	}

	var subcmdInfo *Command
	for i := range c.subcmds {
		if c.subcmds[i].name == p.Subcmd.Name {
			subcmdInfo = &c.subcmds[i]
			break
		}
	}
	if subcmdInfo == nil {
		return UnknownSubcmdError{Name: p.Subcmd.Name}
	}

	errFromSubcmd := parse(subcmdInfo, p.Subcmd, rest[1:])
	if _, ok := errFromSubcmd.(HelpRequestError); ok {
		return errFromSubcmd
	}

	return errMissingOpts
}

func newParsedInput(in *Input, src ParsedFrom, rawValue string) (ParsedInput, error) {
	var val any
	var err error

	switch {
	// If we have a value parser, use that.
	case in.valueParser != nil:
		val, err = in.valueParser(rawValue)
	// If we don't have a value parser but we know it's a boolean option, use the
	// default boolean parser.
	case in.isBoolOpt:
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
		return ParsedInput{}, err
	}

	return ParsedInput{
		ID:       in.id,
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
