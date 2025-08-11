package cli

import (
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
	u.Grow((len(c.Opts) + len(c.Args) + len(c.Subcmds)) * 500)

	helpWriteHeader(&u, c)
	helpWriteUsageLines(&u, c)

	u.WriteString("\noptions:\n")
	opts := slices.Clone(c.Opts)
	slices.SortStableFunc(opts, func(a, b InputInfo) int {
		nameToCmpA := string(a.NameShort)
		nameToCmpB := string(b.NameShort)
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
				desc += " (default: " + o.StrDefault + ")"
			}
			if o.EnvVar != "" {
				desc += " [$" + o.EnvVar + "]"
			}

			content := "  "
			if o.NameShort != 0 {
				content += "-" + string(o.NameShort)
			}
			if o.NameLong != "" {
				if o.NameShort != 0 {
					content += ", "
				}
				content += "--" + o.NameLong
			}
			if an := o.optUsgArgName(); an != "" {
				content += "  " + an
			}

			u.WriteString(content)
			u.WriteString("\n" + strings.Repeat(" ", 6))
			u.WriteString(wrapBlurb(desc, 6, helpMsgTextWidth))
			u.WriteByte('\n')
		}
	} else {
		for i, o := range opts {
			desc := o.HelpBlurb
			if o.IsRequired {
				desc += " (required)"
			}
			if o.HasStrDefault {
				desc += " (default: " + o.StrDefault + ")"
			}
			if o.EnvVar != "" {
				desc += " [$" + o.EnvVar + "]"
			}
			rightPadding := strings.Repeat(" ", optNameColWidth-len(optLeftPaddedNames[i])+3)
			paddedNameAndVal := "  " + optLeftPaddedNames[i] + rightPadding
			u.WriteString(paddedNameAndVal)
			u.WriteString(wrapBlurb(desc, len(paddedNameAndVal), helpMsgTextWidth))
			u.WriteByte('\n')
		}
	}

	if len(c.Args) > 0 {
		u.WriteString("\narguments:\n")

		argNames := make([]string, len(c.Args))
		argNameColWidth := 0
		for i, a := range c.Args {
			argName := a.ID
			if a.ValueName != "" {
				argName = a.ValueName
			}
			if a.IsRequired {
				argName = "<" + argName + ">"
			} else {
				argName = "[" + argName + "]"
			}
			argNames[i] = argName
			if l := len(argName); l > argNameColWidth {
				argNameColWidth = l
			}
		}
		for i, a := range c.Args {
			desc := a.HelpBlurb
			if a.IsRequired {
				desc += " (required)"
			}
			if a.HasStrDefault {
				desc += " (default: " + a.StrDefault + ")"
			}
			if a.EnvVar != "" {
				desc += " [$" + a.EnvVar + "]"
			}

			if argNameColWidth > helpShortMsgMaxFirstColLen {
				u.WriteString("  " + argNames[i])
				u.WriteString("\n" + strings.Repeat(" ", 5))
				u.WriteString(wrapBlurb(desc, 5, helpMsgTextWidth))
			} else {
				rightPadding := strings.Repeat(" ", argNameColWidth-len(argNames[i])+3)
				paddedNameCol := "  " + argNames[i] + rightPadding
				u.WriteString(paddedNameCol)
				u.WriteString(wrapBlurb(desc, len(paddedNameCol), helpMsgTextWidth))
			}
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
			rightPadding := strings.Repeat(" ", maxCmdNameLen-len(c.Subcmds[i].Name)+3)
			paddedNameCol := "   " + c.Subcmds[i].Name + rightPadding
			u.WriteString(paddedNameCol)
			u.WriteString(wrapBlurb(c.Subcmds[i].HelpBlurb, len(paddedNameCol), helpMsgTextWidth) + "\n")
		}
	}

	return u.String()
}

func DefaultFullHelp(c *CommandInfo) string {
	u := strings.Builder{}
	u.Grow((len(c.Opts) + len(c.Args) + len(c.Subcmds)) * 500)

	helpWriteHeader(&u, c)

	if c.HelpExtra != "" {
		u.WriteString("\n\noverview:\n")
		u.WriteString("  " + wrapBlurb(c.HelpExtra, 2, helpMsgTextWidth))
	}

	helpWriteUsageLines(&u, c)

	u.WriteString("\noptions:\n")
	opts := slices.Clone(c.Opts)
	slices.SortStableFunc(opts, func(a, b InputInfo) int {
		nameToCmpA := string(a.NameShort)
		nameToCmpB := string(b.NameShort)
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
			extra += "\n      [default: " + o.StrDefault + "]"
		}
		if o.EnvVar != "" {
			extra += "\n      [env: " + o.EnvVar + "]"
		}

		var usgNamesAndArg string
		{
			if o.NameShort != 0 {
				usgNamesAndArg += "-" + string(o.NameShort)
			}
			if o.NameLong != "" {
				if o.NameShort != 0 {
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
				extra += "\n      [default: " + a.StrDefault + "]"
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
		var maxCmdBlurbLen int
		for i := range c.Subcmds {
			if n := len(c.Subcmds[i].Name); n > maxCmdNameLen {
				maxCmdNameLen = n
			}
			if n := len(c.Subcmds[i].HelpBlurb); n > maxCmdBlurbLen {
				maxCmdBlurbLen = n
			}
		}

		doNonCondensed := maxCmdNameLen > helpShortMsgMaxFirstColLen ||
			maxCmdBlurbLen > (helpMsgTextWidth-maxCmdNameLen-6)

		u.WriteString("\ncommands:\n")
		for i := range c.Subcmds {
			if doNonCondensed {
				u.WriteString("   ")
				u.WriteString(c.Subcmds[i].Name)
				u.WriteString("\n      ")
				u.WriteString(wrapBlurb(c.Subcmds[i].HelpBlurb, 6, helpMsgTextWidth))
				u.WriteByte('\n')
				if i < len(c.Subcmds)-1 {
					u.WriteByte('\n')
				}
			} else {
				rightPadding := strings.Repeat(" ", maxCmdNameLen-len(c.Subcmds[i].Name)+3)
				paddedNameCol := "   " + c.Subcmds[i].Name + rightPadding
				u.WriteString(paddedNameCol + c.Subcmds[i].HelpBlurb + "\n")
			}
		}
	}

	return u.String()
}

func (o *InputInfo) leftPaddedNames() string {
	var s string
	if o.NameShort != 0 {
		s += "-" + string(o.NameShort)
	} else {
		s += "   "
	}

	if o.NameLong != "" {
		if o.NameShort != 0 {
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

func helpWriteHeader(u *strings.Builder, c *CommandInfo) {
	u.WriteString(strings.Join(c.Path, " "))
	if c.HelpBlurb != "" {
		u.WriteString(" - ")
		u.WriteString(c.HelpBlurb)
	}
}

// Build default usage line or use the user provided ones.
func helpWriteUsageLines(u *strings.Builder, c *CommandInfo) {
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
