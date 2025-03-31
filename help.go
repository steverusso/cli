package cli

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
)

type HelpGenerator = func(ParsedInput, *Command) string

func DefaultHelpGenerator(src ParsedInput, c *Command) string {
	if src.From.Opt == "h" {
		return DefaultShortHelp(c)
	}
	return DefaultFullHelp(c)
}

const helpMsgTextWidth = 90

func DefaultShortHelp(c *Command) string {
	u := strings.Builder{}
	u.Grow((len(c.opts) + len(c.args) + len(c.subcmds)) * 200)

	path := strings.Join(c.path, " ")

	u.WriteString(path)
	u.WriteString(" - ")
	u.WriteString(c.helpBlurb)

	// build default usage line or range user provided ones
	u.WriteString("\n\nusage:\n")
	if len(c.helpUsage) == 0 {
		u.WriteString("  ")
		u.WriteString(path)
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
	slices.SortStableFunc(opts, func(a, b Input) int {
		return strings.Compare(a.nameLong, b.nameLong)
	})

	var optNameColWidth int
	for i := range c.opts {
		if l := len(c.opts[i].optUsgNameAndArg()); l > optNameColWidth {
			optNameColWidth = l
		}
	}
	for _, o := range c.opts {
		paddedNameAndArg := fmt.Sprintf("   %-*s   ", optNameColWidth, o.optUsgNameAndArg())
		desc := o.helpBlurb
		if o.hasDefaultValue {
			desc += fmt.Sprintf(" (default: %v)", o.rawDefaultValue)
		}
		if o.env != "" {
			desc += " [$" + o.env + "]"
		}
		content := paddedNameAndArg + wrapBlurb(desc, len(paddedNameAndArg), helpMsgTextWidth)
		u.WriteString(content)
		u.WriteByte('\n')
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
			fmt.Fprintf(&u, "    %-*s   %s\n", maxCmdNameLen, c.subcmds[i].name, c.subcmds[i].helpBlurb)
		}
	}

	return u.String()
}

func DefaultFullHelp(c *Command) string {
	u := strings.Builder{}
	u.Grow((len(c.opts) + len(c.args) + len(c.subcmds)) * 200)

	path := strings.Join(c.path, " ")

	u.WriteString(path)
	u.WriteString(" - ")
	u.WriteString(c.helpBlurb)

	// build default usage line or range user provided ones
	u.WriteString("\n\nusage:\n")
	if len(c.helpUsage) == 0 {
		u.WriteString("  ")
		u.WriteString(path)
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

	if c.helpExtra != "" {
		u.WriteString("\noverview:\n")
		u.WriteString(c.helpExtra)
		u.WriteByte('\n')
	}

	u.WriteString("\noptions:\n")
	opts := slices.Clone(c.opts)
	slices.SortStableFunc(opts, func(a, b Input) int {
		return strings.Compare(a.nameLong, b.nameLong)
	})
	for i, o := range opts {
		var extra string
		if o.hasDefaultValue {
			extra += fmt.Sprintf("\n    [default: %v]", o.rawDefaultValue)
		}
		if o.env != "" {
			extra += "\n    [env: " + o.env + "]"
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
		content += "\n"
		content += "    " + wrapBlurb(o.helpBlurb, 4, helpMsgTextWidth)
		if extra != "" {
			content += "\n" + extra
		}
		if i < len(c.opts)-1 {
			content += "\n"
		}

		u.WriteString(content)
		u.WriteByte('\n')
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
			fmt.Fprintf(&u, "    %-*s   %s\n", maxCmdNameLen, c.subcmds[i].name, c.subcmds[i].helpBlurb)
		}
	}

	return u.String()
}

func (o *Input) optUsgNameAndArg() string {
	var s string
	if o.nameShort != "" {
		s += "-" + o.nameShort
	} else {
		s += "   "
	}

	if o.nameLong != "" {
		if o.nameShort != "" {
			s += ", "
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
func (o *Input) optUsgArgName() string {
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
