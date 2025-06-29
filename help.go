package cli

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
)

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
	u.Grow((len(c.Opts) + len(c.Args) + len(c.Subcmds)) * 200)

	u.WriteString(strings.Join(c.Path, " "))
	u.WriteString(" - ")
	u.WriteString(c.HelpBlurb)

	// build default usage line or range user provided ones
	u.WriteString("\n\nusage:\n")
	if len(c.HelpUsage) == 0 {
		u.WriteString("  ")
		u.WriteString(c.Path[len(c.Path)-1])
		u.WriteString(" [options]")
		switch {
		case len(c.Args) > 0:
			u.WriteString(" [arguments]")
		case len(c.Subcmds) > 0:
			u.WriteString(" <command>")
		}
		u.WriteByte('\n')
	} else {
		for i := range c.HelpUsage {
			u.WriteString("  ")
			u.WriteString(c.HelpUsage[i])
			u.WriteByte('\n')
		}
	}

	u.WriteString("\noptions:\n")
	opts := slices.Clone(c.Opts)
	slices.SortStableFunc(opts, func(a, b InputInfo) int {
		nameToCmpA := a.NameShort
		nameToCmpB := b.NameShort
		if a.NameLong != "" {
			nameToCmpA = a.NameLong
		}
		if b.NameLong != "" {
			nameToCmpB = b.NameLong
		}
		return strings.Compare(nameToCmpA, nameToCmpB)
	})

	// First we need to determine the length of the longest left padded 'name(s) + value
	// name' for the options (meaing which `-s, --long <arg>` is the longest when spacing
	// is added for any absent short names). This will determine if we ouptut 'condensed'
	// option data or not, and (if we do condensed) what the right padding should be for
	// the option name columns that are shorter than the longest one.
	optLeftPaddedNames := make([]string, len(opts))
	optNameColWidth := 0
	for i := range opts {
		optLeftPaddedNames[i] = opts[i].leftPaddedNames()
		if l := len(optLeftPaddedNames[i]); l > optNameColWidth {
			optNameColWidth = l
		}
	}

	// If the name column width would be longer than the (arbitrary) max width, then we'll
	// output 'non-condensed' lines of option data so it won't all look awkwardly crammed
	// off to the right.
	if optNameColWidth > helpShortMsgMaxFirstColLen {
		for _, o := range opts {
			desc := o.HelpBlurb
			if o.IsRequired {
				desc += " (required)"
			}
			if o.HasStrDefault {
				desc += fmt.Sprintf(" (default: %v)", o.StrDefault)
			}
			if o.EnvVar != "" {
				desc += " [$" + o.EnvVar + "]"
			}
			var namesAndVal string
			{
				if o.NameShort != "" {
					namesAndVal += "-" + o.NameShort
				}
				if o.NameLong != "" {
					if o.NameShort != "" {
						namesAndVal += ", "
					}
					namesAndVal += "--" + o.NameLong
				}
				if an := o.optUsgArgName(); an != "" {
					namesAndVal += "  " + an
				}
			}
			content := "  " + namesAndVal
			content += "\n" + strings.Repeat(" ", 6)
			content += wrapBlurb(desc, 6, helpMsgTextWidth)
			u.WriteString(content)
			u.WriteByte('\n')
		}
	} else {
		for i, o := range opts {
			desc := o.HelpBlurb
			if o.IsRequired {
				desc += " (required)"
			}
			if o.HasStrDefault {
				desc += fmt.Sprintf(" (default: %v)", o.StrDefault)
			}
			if o.EnvVar != "" {
				desc += " [$" + o.EnvVar + "]"
			}
			content := fmt.Sprintf("  %-*s   ", optNameColWidth, optLeftPaddedNames[i])
			content += wrapBlurb(desc, len(content), helpMsgTextWidth)
			u.WriteString(content)
			u.WriteByte('\n')
		}
	}

	if len(c.Args) > 0 {
		u.WriteString("\narguments:\n")

		var argNameColWidth int
		for i := range c.Args {
			argName := c.Args[i].ID
			if c.Args[i].ValueName != "" {
				argName = c.Args[i].ValueName
			}
			if l := len(argName); l > argNameColWidth && l <= helpShortMsgMaxFirstColLen {
				argNameColWidth = l
			}
		}
		argNameColWidth += 2
		for _, a := range c.Args {
			argName := a.ID
			if a.ValueName != "" {
				argName = a.ValueName
			}
			if a.IsRequired {
				argName = "<" + argName + ">"
			} else {
				argName = "[" + argName + "]"
			}
			paddedNameAndArg := fmt.Sprintf("  %-*s", argNameColWidth, argName)
			desc := a.HelpBlurb
			if a.IsRequired {
				desc += " (required)"
			}
			if a.HasStrDefault {
				desc += fmt.Sprintf(" (default: %v)", a.StrDefault)
			}
			if a.EnvVar != "" {
				desc += " [$" + a.EnvVar + "]"
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

	if len(c.Subcmds) > 0 {
		var maxCmdNameLen int
		for i := range c.Subcmds {
			if n := len(c.Subcmds[i].Name); n > maxCmdNameLen {
				maxCmdNameLen = n
			}
		}

		u.WriteString("\ncommands:\n")
		for i := range c.Subcmds {
			fmt.Fprintf(&u, "   %-*s   %s\n", maxCmdNameLen, c.Subcmds[i].Name, c.Subcmds[i].HelpBlurb)
		}
	}

	return u.String()
}

func DefaultFullHelp(c *CommandInfo) string {
	u := strings.Builder{}
	u.Grow((len(c.Opts) + len(c.Args) + len(c.Subcmds)) * 200)

	u.WriteString(strings.Join(c.Path, " "))
	u.WriteString(" - ")
	u.WriteString(c.HelpBlurb)

	if c.HelpExtra != "" {
		u.WriteString("\n\noverview:\n")
		u.WriteString("  " + wrapBlurb(c.HelpExtra, 2, helpMsgTextWidth))
	}

	// build default usage line or range user provided ones
	u.WriteString("\n\nusage:\n")
	if len(c.HelpUsage) == 0 {
		u.WriteString("  ")
		u.WriteString(c.Path[len(c.Path)-1])
		u.WriteString(" [options]")
		switch {
		case len(c.Args) > 0:
			u.WriteString(" [arguments]")
		case len(c.Subcmds) > 0:
			u.WriteString(" <command>")
		}
		u.WriteByte('\n')
	} else {
		for i := range c.HelpUsage {
			u.WriteString("  ")
			u.WriteString(c.HelpUsage[i])
			u.WriteByte('\n')
		}
	}

	u.WriteString("\noptions:\n")
	opts := slices.Clone(c.Opts)
	slices.SortStableFunc(opts, func(a, b InputInfo) int {
		nameToCmpA := a.NameShort
		nameToCmpB := b.NameShort
		if a.NameLong != "" {
			nameToCmpA = a.NameLong
		}
		if b.NameLong != "" {
			nameToCmpB = b.NameLong
		}
		return strings.Compare(nameToCmpA, nameToCmpB)
	})
	for i, o := range opts {
		var extra string
		if o.HasStrDefault {
			extra += fmt.Sprintf("\n      [default: %v]", o.StrDefault)
		}
		if o.EnvVar != "" {
			extra += "\n      [env: " + o.EnvVar + "]"
		}

		var usgNamesAndArg string
		{
			if o.NameShort != "" {
				usgNamesAndArg += "-" + o.NameShort
			}
			if o.NameLong != "" {
				if o.NameShort != "" {
					usgNamesAndArg += ", "
				}
				usgNamesAndArg += "--" + o.NameLong
			}
			if an := o.optUsgArgName(); an != "" {
				usgNamesAndArg += "  " + an
			}
		}

		content := "  " + usgNamesAndArg
		if o.IsRequired {
			content += "   (required)"
		}
		if o.HelpBlurb != "" {
			content += "\n      " + wrapBlurb(o.HelpBlurb, 6, helpMsgTextWidth)
		}
		if extra != "" {
			content += "\n" + extra
		}
		if i < len(c.Opts)-1 {
			content += "\n"
		}

		u.WriteString(content)
		u.WriteByte('\n')
	}

	if len(c.Args) > 0 {
		u.WriteString("\narguments:\n")
		for i, a := range c.Args {
			var extra string
			if a.HasStrDefault {
				extra += fmt.Sprintf("\n      [default: %v]", a.StrDefault)
			}
			if a.EnvVar != "" {
				extra += "\n      [env: " + a.EnvVar + "]"
			}

			argName := a.ID
			if a.ValueName != "" {
				argName = a.ValueName
			}
			if a.IsRequired {
				argName = "<" + argName + ">"
			} else {
				argName = "[" + argName + "]"
			}

			content := "  " + argName
			if a.IsRequired {
				content += "   (required)"
			}
			content += "\n"
			content += "      " + wrapBlurb(a.HelpBlurb, 6, helpMsgTextWidth)
			if extra != "" {
				content += "\n" + extra
			}
			if i < len(c.Args)-1 {
				content += "\n"
			}

			u.WriteString(content)
			u.WriteByte('\n')
		}
	}

	if len(c.Subcmds) > 0 {
		var maxCmdNameLen int
		for i := range c.Subcmds {
			if n := len(c.Subcmds[i].Name); n > maxCmdNameLen {
				maxCmdNameLen = n
			}
		}

		u.WriteString("\ncommands:\n")
		for i := range c.Subcmds {
			fmt.Fprintf(&u, "   %-*s   %s\n", maxCmdNameLen, c.Subcmds[i].Name, c.Subcmds[i].HelpBlurb)
		}
	}

	return u.String()
}

func (o *InputInfo) leftPaddedNames() string {
	var s string
	if o.NameShort != "" {
		s += "-" + o.NameShort
	} else {
		s += "   "
	}

	if o.NameLong != "" {
		if o.NameShort != "" {
			s += ", "
		} else {
			s += " "
		}
		s += "--" + o.NameLong
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
	if o.IsBoolOpt {
		return ""
	}
	if o.ValueName != "" {
		return "<" + o.ValueName + ">"
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
