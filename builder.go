package cli

import "strings"

var DefaultHelpInput = NewBoolOpt("help").
	Short('h').
	Help("Show this help message and exit.").
	WithHelpGen(DefaultHelpGenerator)

var (
	errMixingPosArgsAndSubcmds = "commands cannot have both positional args and subcommands"
	errEmptyCmdName            = "empty command name"
	errEmptyInputID            = "inputs must have non-empty, unique ids"
	errEmptyOptNames           = "options must have either a short or long name"
	errOptAsPosArg             = "adding an option as a positional argument"
	errReqArgAfterOptional     = "required positional arguments cannot come after optional ones"
)

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

	// option assertions
	for i := 0; i < len(c.Opts)-1; i++ {
		for z := i + 1; z < len(c.Opts); z++ {
			// assert there are no duplicate input ids
			if c.Opts[i].ID == c.Opts[z].ID {
				panic("command '" + strings.Join(c.Path, " ") +
					"' contains duplicate option ids '" + c.Opts[i].ID + "'")
			}

			// assert there are no duplicate long or short option names
			if c.Opts[i].NameShort != "" && c.Opts[i].NameShort == c.Opts[z].NameShort {
				panic("command '" + strings.Join(c.Path, " ") +
					"' contains duplicate option short name '" + c.Opts[i].NameShort + "'")
			}
			if c.Opts[i].NameLong != "" && c.Opts[i].NameLong == c.Opts[z].NameLong {
				panic("command '" + strings.Join(c.Path, " ") +
					"' contains duplicate option long name '" + c.Opts[i].NameLong + "'")
			}
		}
	}

	// assert there are no duplicate arg ids
	for i := 0; i < len(c.Args)-1; i++ {
		for z := i + 1; z < len(c.Args); z++ {
			if c.Args[i].ID == c.Args[z].ID {
				panic("command '" + strings.Join(c.Path, " ") +
					"' contains duplicate argument ids '" + c.Args[i].ID + "'")
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

func (c CommandInfo) Opt(o InputInfo) CommandInfo {
	// Assert `o` is not a positional arg by making sure it has at least one option name.
	if o.NameShort == "" && o.NameLong == "" {
		panic(errEmptyOptNames)
	}
	c.Opts = append(c.Opts, o)
	return c
}

func (c CommandInfo) Arg(a InputInfo) CommandInfo {
	if len(c.Subcmds) > 0 {
		panic(errMixingPosArgsAndSubcmds)
	}
	// Assert the given input is not an option.
	if a.isOption() {
		panic(errOptAsPosArg)
	}
	// Ensure a required positional arg isn't coming after an optional one.
	if a.IsRequired && len(c.Args) > 0 && !c.Args[len(c.Args)-1].IsRequired {
		panic(errReqArgAfterOptional)
	}

	c.Args = append(c.Args, a)
	return c
}

func (c CommandInfo) Subcmd(sc CommandInfo) CommandInfo {
	if len(c.Args) > 0 {
		panic(errMixingPosArgsAndSubcmds)
	}

	sc.Path = make([]string, len(c.Path))
	copy(sc.Path, c.Path)
	sc.Path = append(sc.Path, sc.Name)

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
	in.NameShort = string(c)
	return in
}

// ShortOnly sets this option's short name to the given character and removes any long
// name it may have had at this point. In order to create an option that has both a short
// and long name, see [InputInfo.Short]. Use [InputInfo.Long] to add a long name back.
func (in InputInfo) ShortOnly(c byte) InputInfo {
	in.NameLong = ""
	return in.Short(c)
}

func (in InputInfo) Long(name string) InputInfo {
	in.NameLong = name
	return in
}

func (in InputInfo) Help(blurb string) InputInfo {
	in.HelpBlurb = blurb
	return in
}

func (in InputInfo) Env(e string) InputInfo {
	in.EnvVar = e
	return in
}

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

func (in InputInfo) Default(v string) InputInfo {
	in.StrDefault = v
	in.HasStrDefault = true
	return in
}

func (in InputInfo) WithHelpGen(hg HelpGenerator) InputInfo {
	in.HelpGen = hg
	return in
}

func (in *InputInfo) isOption() bool {
	return in.NameShort != "" || in.NameLong != ""
}
