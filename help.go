package cli

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
)

type HelpGenerator = func(Input, *CommandInfo) string

// DefaultHelpGenerator will use [DefaultShortHelp] if src is the short option,
// or it'll use [DefaultFullHelp] if the src is the long option name.
func DefaultHelpGenerator(src Input, c *CommandInfo) string {
	if len(src.From.Opt) == 1 {
		return DefaultShortHelp(c)
	}
	return DefaultFullHelp(c)
}

const (
	helpMsgTextWidth           = 90
	helpShortMsgMaxFirstColLen = 24
)

func DefaultShortHelp(c *CommandInfo) string {
	u := strings.Builder{}
	u.Grow((len(c.opts) + len(c.args) + len(c.subcmds)) * 200)

	u.WriteString(strings.Join(c.path, " "))
	u.WriteString(" - ")
	u.WriteString(c.helpBlurb)

	// build default usage line or range user provided ones
	u.WriteString("\n\nusage:\n")
	if len(c.helpUsage) == 0 {
		u.WriteString("  ")
		u.WriteString(c.path[len(c.path)-1])
		u.WriteString(" [options]")
		switch {
		case len(c.args) > 0:
			u.WriteString(" [arguments]")
		case len(c.subcmds) > 0:
			u.WriteString(" <command>")
		}
		u.WriteByte('\n')
	} else {
		for i := range c.helpUsage {
			u.WriteString("  ")
			u.WriteString(c.helpUsage[i])
			u.WriteByte('\n')
		}
	}

	u.WriteString("\noptions:\n")
	opts := slices.Clone(c.opts)
	slices.SortStableFunc(opts, func(a, b InputInfo) int {
		nameToCmpA := a.nameShort
		nameToCmpB := b.nameShort
		if a.nameLong != "" {
			nameToCmpA = a.nameLong
		}
		if b.nameLong != "" {
			nameToCmpB = b.nameLong
		}
		return strings.Compare(nameToCmpA, nameToCmpB)
	})

	var optNameColWidth int
	for i := range c.opts {
		if l := len(c.opts[i].optUsgNameAndArg()); l > optNameColWidth && l <= helpShortMsgMaxFirstColLen {
			optNameColWidth = l
		}
	}
	for _, o := range opts {
		paddedNameAndArg := fmt.Sprintf("  %-*s", optNameColWidth, o.optUsgNameAndArg())
		desc := o.helpBlurb
		if o.isRequired {
			desc += " (required)"
		}
		if o.hasDefaultValue {
			desc += fmt.Sprintf(" (default: %v)", o.rawDefaultValue)
		}
		if o.env != "" {
			desc += " [$" + o.env + "]"
		}
		content := paddedNameAndArg
		if len(paddedNameAndArg) > helpShortMsgMaxFirstColLen {
			content += "\n" + strings.Repeat(" ", optNameColWidth+5)
		} else {
			content += "   "
		}
		content += wrapBlurb(desc, len(paddedNameAndArg)+3, helpMsgTextWidth)
		u.WriteString(content)
		u.WriteByte('\n')
	}

	if len(c.args) > 0 {
		u.WriteString("\narguments:\n")

		var argNameColWidth int
		for i := range c.args {
			argName := c.args[i].id
			if c.args[i].valueName != "" {
				argName = c.args[i].valueName
			}
			if l := len(argName); l > argNameColWidth && l <= helpShortMsgMaxFirstColLen {
				argNameColWidth = l
			}
		}
		argNameColWidth += 2
		for _, a := range c.args {
			argName := a.id
			if a.valueName != "" {
				argName = a.valueName
			}
			if a.isRequired {
				argName = "<" + argName + ">"
			} else {
				argName = "[" + argName + "]"
			}
			paddedNameAndArg := fmt.Sprintf("  %-*s", argNameColWidth, argName)
			desc := a.helpBlurb
			if a.isRequired {
				desc += " (required)"
			}
			if a.hasDefaultValue {
				desc += fmt.Sprintf(" (default: %v)", a.rawDefaultValue)
			}
			if a.env != "" {
				desc += " [$" + a.env + "]"
			}
			content := paddedNameAndArg
			if len(paddedNameAndArg) > helpShortMsgMaxFirstColLen {
				content += "\n" + strings.Repeat(" ", argNameColWidth+5)
			} else {
				content += "   "
			}
			content += wrapBlurb(desc, len(paddedNameAndArg)+3, helpMsgTextWidth)
			u.WriteString(content)
			u.WriteByte('\n')
		}
	}

	if len(c.subcmds) > 0 {
		var maxCmdNameLen int
		for i := range c.subcmds {
			if n := len(c.subcmds[i].name); n > maxCmdNameLen {
				maxCmdNameLen = n
			}
		}

		u.WriteString("\ncommands:\n")
		for i := range c.subcmds {
			fmt.Fprintf(&u, "   %-*s   %s\n", maxCmdNameLen, c.subcmds[i].name, c.subcmds[i].helpBlurb)
		}
	}

	return u.String()
}

func DefaultFullHelp(c *CommandInfo) string {
	u := strings.Builder{}
	u.Grow((len(c.opts) + len(c.args) + len(c.subcmds)) * 200)

	u.WriteString(strings.Join(c.path, " "))
	u.WriteString(" - ")
	u.WriteString(c.helpBlurb)

	if c.helpExtra != "" {
		u.WriteString("\n\noverview:\n")
		u.WriteString("  " + wrapBlurb(c.helpExtra, 2, helpMsgTextWidth))
	}

	// build default usage line or range user provided ones
	u.WriteString("\n\nusage:\n")
	if len(c.helpUsage) == 0 {
		u.WriteString("  ")
		u.WriteString(c.path[len(c.path)-1])
		u.WriteString(" [options]")
		switch {
		case len(c.args) > 0:
			u.WriteString(" [arguments]")
		case len(c.subcmds) > 0:
			u.WriteString(" <command>")
		}
		u.WriteByte('\n')
	} else {
		for i := range c.helpUsage {
			u.WriteString("  ")
			u.WriteString(c.helpUsage[i])
			u.WriteByte('\n')
		}
	}

	u.WriteString("\noptions:\n")
	opts := slices.Clone(c.opts)
	slices.SortStableFunc(opts, func(a, b InputInfo) int {
		return strings.Compare(a.nameLong, b.nameLong)
	})
	for i, o := range opts {
		var extra string
		if o.hasDefaultValue {
			extra += fmt.Sprintf("\n      [default: %v]", o.rawDefaultValue)
		}
		if o.env != "" {
			extra += "\n      [env: " + o.env + "]"
		}

		var usgNamesAndArg string
		{
			if o.nameShort != "" {
				usgNamesAndArg += "-" + o.nameShort
			}

			if o.nameLong != "" {
				if o.nameShort != "" {
					usgNamesAndArg += ", "
				}
				usgNamesAndArg += "--" + o.nameLong
			}

			if an := o.optUsgArgName(); an != "" {
				usgNamesAndArg += "  " + an
			}
		}

		content := "  " + usgNamesAndArg
		if o.isRequired {
			content += "   (required)"
		}
		if o.helpBlurb != "" {
			content += "\n      " + wrapBlurb(o.helpBlurb, 6, helpMsgTextWidth)
		}
		if extra != "" {
			content += "\n" + extra
		}
		if i < len(c.opts)-1 {
			content += "\n"
		}

		u.WriteString(content)
		u.WriteByte('\n')
	}

	if len(c.args) > 0 {
		u.WriteString("\narguments:\n")
		for i, a := range c.args {
			var extra string
			if a.hasDefaultValue {
				extra += fmt.Sprintf("\n      [default: %v]", a.rawDefaultValue)
			}
			if a.env != "" {
				extra += "\n      [env: " + a.env + "]"
			}

			argName := a.id
			if a.valueName != "" {
				argName = a.valueName
			}
			if a.isRequired {
				argName = "<" + argName + ">"
			} else {
				argName = "[" + argName + "]"
			}

			content := "  " + argName
			if a.isRequired {
				content += "   (required)"
			}
			content += "\n"
			content += "      " + wrapBlurb(a.helpBlurb, 6, helpMsgTextWidth)
			if extra != "" {
				content += "\n" + extra
			}
			if i < len(c.args)-1 {
				content += "\n"
			}

			u.WriteString(content)
			u.WriteByte('\n')
		}
	}

	if len(c.subcmds) > 0 {
		var maxCmdNameLen int
		for i := range c.subcmds {
			if n := len(c.subcmds[i].name); n > maxCmdNameLen {
				maxCmdNameLen = n
			}
		}

		u.WriteString("\ncommands:\n")
		for i := range c.subcmds {
			fmt.Fprintf(&u, "   %-*s   %s\n", maxCmdNameLen, c.subcmds[i].name, c.subcmds[i].helpBlurb)
		}
	}

	return u.String()
}

func (o *InputInfo) optUsgNameAndArg() string {
	var s string
	if o.nameShort != "" {
		s += "-" + o.nameShort
	} else {
		s += "   "
	}

	if o.nameLong != "" {
		if o.nameShort != "" {
			s += ", "
		} else {
			s += " "
		}
		s += "--" + o.nameLong
	}

	if an := o.optUsgArgName(); an != "" {
		s += "  " + an
	}
	return s
}

// optUsgArgName returns the usage text of an option argument for non-boolean options. For
// example, if there's a string option named `file`, the usage might look something like
// `--file <arg>` where "<arg>" is the usage argument name text.
func (o *InputInfo) optUsgArgName() string {
	if o.isBoolOpt {
		return ""
	}
	if o.valueName != "" {
		return "<" + o.valueName + ">"
	}
	return "<arg>"
}

func wrapBlurb(v string, indentLen, lineLen int) string {
	s := wrapText(v, indentLen, lineLen)
	return s[indentLen:]
}

type wordWrapper struct {
	indent string
	word   strings.Builder
	line   strings.Builder
	result strings.Builder
}

func wrapText(v string, indentLen, lineLen int) string {
	var ww wordWrapper
	ww.indent = strings.Repeat(" ", indentLen)
	ww.word.Grow(lineLen)
	ww.line.Grow(lineLen)
	ww.line.WriteString(ww.indent)
	ww.result.Grow(len(v))

	for _, c := range strings.TrimSpace(v) {
		if !unicode.IsSpace(c) {
			ww.word.WriteRune(c)
			continue
		}
		if c == '\n' {
			ww.takeWordAndReset()
			ww.takeLineAndReset()
			continue
		}
		if ww.line.Len()+ww.word.Len() > lineLen {
			ww.takeLineAndReset()
		}
		ww.takeWordAndReset()
		ww.line.WriteRune(c)
	}
	if ww.word.Len() > 0 {
		if ww.line.Len()+ww.word.Len() > lineLen {
			ww.takeLineAndReset()
		}
		ww.takeWordAndReset()
	}
	if ww.line.Len() > 0 {
		ww.result.WriteString(ww.line.String())
		ww.line.Reset()
	}

	res := ww.result.String()
	ww.result.Reset()
	return res
}

func (ww *wordWrapper) takeWordAndReset() {
	ww.line.WriteString(ww.word.String())
	ww.word.Reset()
}

func (ww *wordWrapper) takeLineAndReset() {
	ln := strings.TrimRightFunc(ww.line.String(), unicode.IsSpace) // remove trailing whitespace
	ww.result.WriteString(ln)
	ww.result.WriteRune('\n')
	ww.line.Reset()
	ww.line.WriteString(ww.indent)
}
