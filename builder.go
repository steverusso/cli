package cli

import (
	"runtime/debug"
	"slices"
	"strings"
)

var DefaultHelpInput = NewBoolOpt("help").
	Short('h').
	Help("Show this help message and exit.").
	WithHelpGen(DefaultHelpGenerator)

var DefaultVersionOpt = NewVersionOpt('v', "version", VersionOptConfig{
	HelpBlurb: "Print the build info version and exit",
})

var (
	errMixingPosArgsAndSubcmds = "commands cannot have both positional args and subcommands"
	errEmptyCmdName            = "empty command name"
	errEmptyInputID            = "inputs must have non-empty, unique ids"
	errEmptyOptNames           = "options must have either a short or long name"
	errOptAsPosArg             = "adding an option as a positional argument"
	errReqArgAfterOptional     = "required positional arguments cannot come after optional ones"
)

// New is intented to initialize a new root command. If no name is provided, it will use
// runtime information to get the module name and use that for the command's name.
// Anything more than a single name provided is ignored.
func New(name ...string) CommandInfo {
	var cmdName string
	if len(name) > 0 {
		cmdName = name[0]
	} else {
		info, ok := debug.ReadBuildInfo()
		if !ok {
			panic("failed to read build info")
		}
		lastSlash := strings.LastIndexByte(info.Path, '/')
		cmdName = info.Path[lastSlash+1:]
	}

	return NewCmd(cmdName)
}

func (c *CommandInfo) prepareAndValidate() {
	// Add the default help option here as long as this
	// command doesn't already have a help option.
	var hasHelpOpt bool
	for i := range c.Opts {
		if c.Opts[i].HelpGen != nil {
			hasHelpOpt = true
			break
		}
	}
	if !hasHelpOpt {
		*c = (*c).Opt(DefaultHelpInput)
	}

	// assert there are no duplicate input ids across the options and positional arguments
	inputIDs := make([]string, 0, len(c.Opts)+len(c.Args))
	for i := range len(c.Opts) {
		id := c.Opts[i].ID
		if slices.Contains(inputIDs, id) {
			panic("command '" + strings.Join(c.Path, " ") +
				"' contains duplicate input ids '" + id + "'")
		}
		inputIDs = append(inputIDs, id)
	}
	for i := range len(c.Args) {
		id := c.Args[i].ID
		if slices.Contains(inputIDs, id) {
			panic("command '" + strings.Join(c.Path, " ") +
				"' contains duplicate input ids '" + id + "'")
		}
		inputIDs = append(inputIDs, id)
	}

	// option assertions
	for i := 0; i < len(c.Opts)-1; i++ {
		for z := i + 1; z < len(c.Opts); z++ {
			// assert there are no duplicate long or short option names
			if c.Opts[i].NameShort != 0 && c.Opts[i].NameShort == c.Opts[z].NameShort {
				panic("command '" + strings.Join(c.Path, " ") +
					"' contains duplicate option short name '" + string(c.Opts[i].NameShort) + "'")
			}
			if c.Opts[i].NameLong != "" && c.Opts[i].NameLong == c.Opts[z].NameLong {
				panic("command '" + strings.Join(c.Path, " ") +
					"' contains duplicate option long name '" + c.Opts[i].NameLong + "'")
			}
		}
	}

	// subcommand names must be unique across Subcmds
	for i := 0; i < len(c.Subcmds)-1; i++ {
		for z := i + 1; z < len(c.Subcmds); z++ {
			if c.Subcmds[i].Name == c.Subcmds[z].Name {
				panic("command '" + strings.Join(c.Path, " ") +
					"' contains duplicate subcommand name '" + c.Subcmds[i].Name + "'")
			}
		}
	}
	for i := range c.Subcmds {
		c.Subcmds[i].Path = make([]string, len(c.Path)+1)
		copy(c.Subcmds[i].Path, c.Path)
		c.Subcmds[i].Path[len(c.Subcmds[i].Path)-1] = c.Subcmds[i].Name

		c.Subcmds[i].prepareAndValidate()
	}
}

func NewCmd(name string) CommandInfo {
	// assert command name isn't empty and doesn't contain any whitespace
	if name == "" {
		panic(errEmptyCmdName)
	}
	for i := range name {
		switch name[i] {
		case ' ', '\t', '\n', '\r':
			panic("invalid command name '" + name + "': cannot contain whitespace")
		}
	}

	return CommandInfo{
		Name: name,
		Path: []string{name},
		Opts: make([]InputInfo, 0, 5),
	}
}

func (c CommandInfo) Help(blurb string) CommandInfo {
	c.HelpBlurb = blurb
	return c
}

// ExtraHelp adds an "overview" section to the Command's help message. This is typically
// for longer-form content that wouldn't fit well within the 1-2 sentence "blurb."
func (c CommandInfo) ExtraHelp(extra string) CommandInfo {
	c.HelpExtra = extra
	return c
}

// Usage overrides the default "usage" lines in the command's help message. These are
// intended to show the user some different ways to invoke this command using whatever
// combinations of options / arguments / subcommands.
func (c CommandInfo) Usage(lines ...string) CommandInfo {
	c.HelpUsage = append(c.HelpUsage, lines...)
	return c
}

// SubcmdOptional sets the IsSubcmdOptional field of this CommandInfo to true.
// See that field's documentation to learn more about how it is used.
func (c CommandInfo) SubcmdOptional() CommandInfo {
	c.IsSubcmdOptional = true
	return c
}

// Opt adds o as an option to this CommandInfo. This method will panic if the option has
// neither a long or short name set (this should never happen when using the builder
// pattern starting with the [NewOpt] function or its siblings).
func (c CommandInfo) Opt(o InputInfo) CommandInfo {
	// Assert `o` is not a positional arg by making sure it has at least one option name.
	if o.NameShort == 0 && o.NameLong == "" {
		panic(errEmptyOptNames)
	}
	c.Opts = append(c.Opts, o)
	return c
}

// Arg adds pa as a positional argument to this CommandInfo. This method will panic if this
// command already has one or more subcommands (because positional arguments and
// subcommands are mutually exclusive), or if pa has any option names set, or if pa is
// required but any previously positional argument is not required (because required
// positional arguments cannot come after optional ones).
func (c CommandInfo) Arg(pa InputInfo) CommandInfo {
	if len(c.Subcmds) > 0 {
		panic(errMixingPosArgsAndSubcmds)
	}
	// Assert the given input is not an option.
	if pa.isOption() {
		panic(errOptAsPosArg)
	}
	// Ensure a required positional arg isn't coming after an optional one.
	if pa.IsRequired && len(c.Args) > 0 && !c.Args[len(c.Args)-1].IsRequired {
		panic(errReqArgAfterOptional)
	}

	c.Args = append(c.Args, pa)
	return c
}

// Subcmd adds the given [CommandInfo] sc as a subcommand under c. This function will
// panic if c already has at least one positional argument because commands cannot contain
// both positional arguments and subcommands simultaneously.
func (c CommandInfo) Subcmd(sc CommandInfo) CommandInfo {
	if len(c.Args) > 0 {
		panic(errMixingPosArgsAndSubcmds)
	}
	c.Subcmds = append(c.Subcmds, sc)
	return c
}

// NewOpt returns a new non-boolean option with no parser, which means it will just
// receive the raw string of any provided value. If id is more than a single character
// long, it will be this option's long name. If id is only a single character, it will be
// this option's short name instead. In either case, the long name can be reset using the
// [InputInfo.Long] method.
func NewOpt(id string) InputInfo {
	if id == "" {
		panic(errEmptyInputID)
	}
	if len(id) == 1 {
		return InputInfo{ID: id}.ShortOnly(id[0])
	}
	return InputInfo{ID: id}.Long(id)
}

// NewBoolOpt returns a new boolean option. If no value is provided to this option when
// parsing, it will have a "parsed" value of true. If any value is provided, the
// [ParseBool] value parser is used. Any other parser set by the user for this option will
// be ignored.
func NewBoolOpt(id string) InputInfo {
	o := NewOpt(id)
	o.IsBoolOpt = true
	return o
}

// NewIntOpt returns a new option that uses the [ParseInt] value parser.
func NewIntOpt(id string) InputInfo {
	return NewOpt(id).WithParser(ParseInt)
}

// NewUintOpt returns a new option that uses the [ParseUint] value parser.
func NewUintOpt(id string) InputInfo {
	return NewOpt(id).WithParser(ParseUint)
}

// NewFloat32Opt returns a new option that uses the [ParseFloat32] value parser.
func NewFloat32Opt(id string) InputInfo {
	return NewOpt(id).WithParser(ParseFloat32)
}

// NewFloat64Opt returns a new option that uses the [ParseFloat64] value parser.
func NewFloat64Opt(id string) InputInfo {
	return NewOpt(id).WithParser(ParseFloat64)
}

// NewArg returns a new positional argument input. By default, the arg's display name will
// be the provided id, but this can be overidden with [InputInfo.WithValueName] method.
func NewArg(id string) InputInfo {
	if id == "" {
		panic(errEmptyInputID)
	}
	return InputInfo{ID: id, ValueName: id}
}

// WithParser sets the InputInfo's parser to the given [ValueParser]. This will override any
// parser that has been set up until this point. Providing nil as the parser will restore
// the default behavior of just using the plain string value when this InputInfo is parsed.
func (in InputInfo) WithParser(vp ValueParser) InputInfo {
	in.ValueParser = vp
	return in
}

// Short sets this option's short name to the given character. In order to create an
// option that has a short name but no long name, see [InputInfo.ShortOnly].
func (in InputInfo) Short(c byte) InputInfo {
	in.NameShort = c
	return in
}

// ShortOnly sets this option's short name to the given character and removes any long
// name it may have had at this point. In order to create an option that has both a short
// and long name, see [InputInfo.Short]. Use [InputInfo.Long] to add a long name back.
func (in InputInfo) ShortOnly(c byte) InputInfo {
	in.NameLong = ""
	return in.Short(c)
}

// Long sets the option long name to the given name. Since an option's long name will be
// the input ID by default, this method is really only necessary when the long name must
// differ from the input ID.
func (in InputInfo) Long(name string) InputInfo {
	in.NameLong = name
	return in
}

// Help sets the brief help blurb for this option or positional argument.
func (in InputInfo) Help(blurb string) InputInfo {
	in.HelpBlurb = blurb
	return in
}

// Env sets the name of the environment variable for this InputInfo. The parser will parse
// the value of that environment variable for this input if it is set.
func (in InputInfo) Env(e string) InputInfo {
	in.EnvVar = e
	return in
}

// Required marks this InputInfo as required, which means an error will be returned when
// parsing if a value is not provided. If this is a positional argument, it must be added
// to a command before any optional positional arguments. Required options, however, can
// be added in any order. See the "Command Line Syntax" section at the top of the docs
func (in InputInfo) Required() InputInfo {
	in.IsRequired = true
	return in
}

// WithValueName sets the display name of this InputInfo's argument value. For non-boolean
// options, it's the argument of the option. For positional arguments, it's the argument
// name itself.
func (in InputInfo) WithValueName(name string) InputInfo {
	in.ValueName = name
	return in
}

// Default sets v as the default string value for this InputInfo, which will be gathered and
// parsed using this InputInfo's parser before any CLI arguments or environment variables.
// This will always happen as the first step in parsing a command, so if a default value
// is set here, then at least it will always be present meaning it's safe to use
// [InputInfo.Get] to get its parsed value.
func (in InputInfo) Default(v string) InputInfo {
	in.StrDefault = v
	in.HasStrDefault = true
	return in
}

// WithHelpGen sets the HelpGen field of this input. See the HelpGen field
// documentation on [InputInfo] to learn more about how it is used.
func (in InputInfo) WithHelpGen(hg HelpGenerator) InputInfo {
	in.HelpGen = hg
	return in
}

// WithVersioner will set this input's Versioner to the given [Versioner]. This will turn
// this input into one that, similar to help inputs, causes the parsing to return a
// [HelpOrVersionRequested] error. See [NewVersionOpt] for a convenient way to create
// version inputs.
func (in InputInfo) WithVersioner(ver Versioner) InputInfo {
	in.Versioner = ver
	return in
}

// VersionOptConfig is used to pass customization values to [NewVersionOpt].
type VersionOptConfig struct {
	HelpBlurb        string
	IncludeGoVersion bool
}

// NewVersionOpt returns a input that will the Versioner field set to a function that
// outputs information based on the given configuration values. At a minimum, the default
// version message will always contain the Go module version obtained from
// [debug.BuildInfo]. Version inputs, similar to help inputs, cause this library's
// parsing to return a [HelpOrVersionRequested] error. This function will panic if the
// given long name is empty and the given short name is either 0 or '-'.
func NewVersionOpt(short byte, long string, cfg VersionOptConfig) InputInfo {
	if cfg.HelpBlurb == "" {
		cfg.HelpBlurb = "Print version info and exit."
	}

	hasShort := short != 0 && short != '-'

	id := long
	if id == "" {
		if !hasShort {
			panic("must provide at least either a long or short name for the version option")
		}
		id = string(short)
	}

	in := NewBoolOpt(id).Help(cfg.HelpBlurb)
	if hasShort && long != "" {
		in = in.Short(short)
	}
	return in.WithVersioner(func(_ Input) string {
		bi, ok := debug.ReadBuildInfo()
		if !ok {
			Fatal(1, "unable to read build info")
		}
		ver := bi.Main.Version + "\n"
		if cfg.IncludeGoVersion {
			ver += bi.GoVersion + "\n"
		}
		return ver
	})
}

func (in *InputInfo) isOption() bool {
	return in.NameShort != 0 || in.NameLong != ""
}
