package cli

import "strings"

var DefaultHelpInput = NewBoolOpt("help").
	Short('h').
	Help("Show this help message and exit.").
	HelpGen(DefaultHelpGenerator)

var (
	errMixingPosArgsAndSubcmds = "commands cannot have both positional args and subcommands"
	errEmptyCmdName            = invalidCmdNameError{}
	errEmptyInputID            = "inputs must have non-empty, unique ids"
	errEmptyOptNames           = "options must have either a short or long name"
	errOptAsPosArg             = "adding an option as a positional argument"
	errReqArgAfterOptional     = "required positional arguments cannot come after optional ones"
)

func (c Command) Build() *RootCommand {
	c.validate()
	return &RootCommand{c: c}
}

func (c *Command) validate() {
	// option assertions
	for i := 0; i < len(c.opts)-1; i++ {
		for z := i + 1; z < len(c.opts); z++ {
			// assert there are no duplicate input ids
			if c.opts[i].id == c.opts[z].id {
				panic(illegalDupError{
					cmdPath: strings.Join(c.path, " "),
					what:    "ids",
					dups:    c.opts[i].id,
				})
			}

			// assert there are no duplicate long or short option names
			if c.opts[i].nameShort != "" && c.opts[i].nameShort == c.opts[z].nameShort {
				panic(illegalDupError{
					cmdPath: strings.Join(c.path, " "),
					what:    "option short names",
					dups:    c.opts[i].nameShort,
				})
			}
			if c.opts[i].nameLong != "" && c.opts[i].nameLong == c.opts[z].nameLong {
				panic(illegalDupError{
					cmdPath: strings.Join(c.path, " "),
					what:    "option long names",
					dups:    c.opts[i].nameLong,
				})
			}
		}
	}

	// assert there are no duplicate arg ids
	for i := 0; i < len(c.args)-1; i++ {
		for z := i + 1; z < len(c.args); z++ {
			if c.args[i].id == c.args[z].id {
				panic(illegalDupError{
					cmdPath: strings.Join(c.path, " "),
					what:    "ids",
					dups:    c.args[i].id,
				})
			}
		}
	}

	// subcommand names must be unique across subcmds
	for i := 0; i < len(c.subcmds)-1; i++ {
		for z := i + 1; z < len(c.subcmds); z++ {
			if c.subcmds[i].name == c.subcmds[z].name {
				panic(illegalDupError{
					cmdPath: strings.Join(c.path, " "),
					what:    "subcommand names",
					dups:    c.subcmds[i].name,
				})
			}
		}
	}

	for i := range c.subcmds {
		c.subcmds[i].validate()
	}
}

type illegalDupError struct {
	cmdPath string
	what    string // input ids, option short names, option long names, subcmd names
	dups    string
}

func (e illegalDupError) String() string {
	return "command '" + e.cmdPath + "' contains options with duplicate " + e.what + " '" + e.dups + "'"
}

type invalidCmdNameError struct {
	name   string
	reason string
}

func (e invalidCmdNameError) String() string {
	if e.name == "" {
		return "empty command name"
	}
	return "invalid command name '" + e.name + "': " + e.reason
}

func NewCmd(name string) Command {
	// assert command name isn't empty and doesn't contain any whitespace
	if name == "" {
		panic(errEmptyCmdName)
	}
	for i := range name {
		switch name[i] {
		case ' ', '\t', '\n', '\r':
			panic(invalidCmdNameError{
				name:   name,
				reason: "cannot contain whitespace",
			})
		}
	}

	c := Command{
		name: name,
		path: []string{name},
		opts: make([]Input, 0, 5),
	}
	return c.Opt(DefaultHelpInput)
}

func (c Command) Help(blurb string) Command {
	c.helpBlurb = blurb
	return c
}

// HelpExtra adds an "overview" section to the Command's help message. This is typically
// for longer-form content that wouldn't fit well within the 1-2 sentence "blurb."
func (c Command) HelpExtra(extra string) Command {
	c.helpExtra = extra
	return c
}

// HelpUsage overrides the default "usage" lines in the command's help message. These are
// intended to show the user some different ways to invoke this command using whatever
// combinations of options / arguments / subcommands.
func (c Command) HelpUsage(lines ...string) Command {
	c.helpUsage = append(c.helpUsage, lines...)
	return c
}

func (c Command) Opt(o Input) Command {
	// Assert `o` is not a positional arg by making sure it has at least one option name.
	if o.nameShort == "" && o.nameLong == "" {
		panic(errEmptyOptNames)
	}
	c.opts = append(c.opts, o)
	return c
}

func (c Command) Arg(a Input) Command {
	if len(c.subcmds) > 0 {
		panic(errMixingPosArgsAndSubcmds)
	}
	// Assert the given input is not an option.
	if a.isOption() {
		panic(errOptAsPosArg)
	}
	// Ensure a required positional arg isn't coming after an optional one.
	if a.isRequired && len(c.args) > 0 && !c.args[len(c.args)-1].isRequired {
		panic(errReqArgAfterOptional)
	}

	c.args = append(c.args, a)
	return c
}

func (c Command) Subcmd(sc Command) Command {
	if len(c.args) > 0 {
		panic(errMixingPosArgsAndSubcmds)
	}

	sc.path = append([]string(nil), c.path...)
	sc.path = append(sc.path, sc.name)

	c.subcmds = append(c.subcmds, sc)
	return c
}

// NewOpt returns a new non-boolean option with no parser, which means it will just
// receive the raw string of any provided value. If id is more than a single character
// long, it will be this option's long name. If id is only a single character, it will be
// this option's short name instead. In either case, the long name can be reset using the
// [Input.Long] method.
func NewOpt(id string) Input {
	if id == "" {
		panic(errEmptyInputID)
	}
	if len(id) == 1 {
		return Input{id: id}.ShortOnly(id[0])
	}
	return Input{id: id}.Long(id)
}

// NewBoolOpt returns a new boolean option. If no value is provided to this option when
// parsing, it will have a "parsed" value of true. If any value is provided, the
// [ParseBool] value parser is used. Any other parser set by the user for this option will
// be ignored.
func NewBoolOpt(id string) Input {
	o := NewOpt(id)
	o.isBoolOpt = true
	return o
}

// NewIntOpt returns a new option that uses the [ParseInt] value parser.
func NewIntOpt(id string) Input {
	return NewOpt(id).WithParser(ParseInt)
}

// NewUintOpt returns a new option that uses the [ParseUint] value parser.
func NewUintOpt(id string) Input {
	return NewOpt(id).WithParser(ParseUint)
}

// NewFloat32Opt returns a new option that uses the [ParseFloat32] value parser.
func NewFloat32Opt(id string) Input {
	return NewOpt(id).WithParser(ParseFloat32)
}

// NewFloat64Opt returns a new option that uses the [ParseFloat64] value parser.
func NewFloat64Opt(id string) Input {
	return NewOpt(id).WithParser(ParseFloat64)
}

// NewArg returns a new positional argument input. By default, the arg's display name will
// be the provided id, but this can be overidden with [Input.ValueName] method.
func NewArg(id string) Input {
	if id == "" {
		panic(errEmptyInputID)
	}
	return Input{id: id, valueName: id}
}

// WithParser sets the Input's parser to the given [ValueParser]. This will override any
// parser that has been set up until this point. Providing nil as the parser will restore
// the default behavior of just using the plain string value when this Input is parsed.
func (in Input) WithParser(vp ValueParser) Input {
	in.valueParser = vp
	return in
}

// Short sets this option's short name to the given character. In order to create an
// option that has a short name but no long name, see [Input.ShortOnly].
func (in Input) Short(c byte) Input {
	in.nameShort = string(c)
	return in
}

// ShortOnly sets this option's short name to the given character and removes any long
// name it may have had at this point. In order to create an option that has both a short
// and long name, see [Input.Short]. Use [Input.Long] to add a long name back.
func (in Input) ShortOnly(c byte) Input {
	in.nameLong = ""
	return in.Short(c)
}

func (in Input) Long(name string) Input {
	in.nameLong = name
	return in
}

func (in Input) Help(blurb string) Input {
	in.helpBlurb = blurb
	return in
}

func (in Input) Env(e string) Input {
	in.env = e
	return in
}

func (in Input) Required() Input {
	in.isRequired = true
	return in
}

// ValueName sets the display name of this Input's argument value. For non-boolean
// options, it's the argument of the option. For positional arguments, it's the argument
// name itself.
func (in Input) ValueName(name string) Input {
	in.valueName = name
	return in
}

func (in Input) Default(v string) Input {
	in.rawDefaultValue = v
	in.hasDefaultValue = true
	return in
}

func (in Input) HelpGen(hg HelpGenerator) Input {
	in.helpGen = hg
	return in
}

func (in *Input) isOption() bool {
	return in.nameShort != "" || in.nameLong != ""
}
