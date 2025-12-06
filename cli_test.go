package cli

import (
	"errors"
	"fmt"
	"image"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestParsing(t *testing.T) {
	type testInputOutput struct {
		Case     string
		envs     map[string]string
		args     []string
		expected Command
		expErr   error
	}
	type testCase struct {
		name       string
		cmd        CommandInfo
		variations []testInputOutput
	}

	for _, tt := range []testCase{
		{
			// no positional args or subcommands
			// errors for missing required opts
			name: "options_only",
			cmd: NewCmd("optsonly").
				Opt(NewBoolOpt("aa")).
				Opt(NewOpt("bb").ShortOnly('b')).
				Opt(NewOpt("cc").Required()).
				Opt(NewOpt("dd").Default("v4")).
				Opt(NewOpt("ee")),
			variations: []testInputOutput{
				{
					Case: ttCase(),
					args: []string{"-b", "v2", "--aa", "--cc=v3"},
					expected: Command{
						Inputs: []Input{
							{ID: "dd", From: ParsedFrom{Default: true}, RawValue: "v4", Value: "v4"},
							{ID: "bb", From: ParsedFrom{Opt: "b"}, RawValue: "v2", Value: "v2"},
							{ID: "aa", From: ParsedFrom{Opt: "aa"}, RawValue: "", Value: true},
							{ID: "cc", From: ParsedFrom{Opt: "cc"}, RawValue: "v3", Value: "v3"},
						},
					},
				}, {
					Case:   ttCase(),
					args:   []string{"-b", "v2", "--aa"},
					expErr: MissingOptionsError{Names: []string{"--cc"}},
				}, {
					Case:   ttCase(),
					args:   []string{"-z"},
					expErr: UnknownOptionError{Name: "-z"},
				}, {
					Case:   ttCase(),
					args:   []string{"--zz=abc"},
					expErr: UnknownOptionError{Name: "--zz=abc"},
				}, {
					Case:   ttCase(),
					args:   []string{"--bb", "B"},
					expErr: UnknownOptionError{Name: "--bb"},
				}, {
					Case:   ttCase(),
					args:   []string{"--dd"},
					expErr: MissingOptionValueError{Name: "dd"},
				}, {
					Case:   ttCase(),
					args:   []string{"-b"},
					expErr: MissingOptionValueError{Name: "b"},
				},
			},
		}, {
			// all provided parsers with defaults
			name: "provided_parsers",
			cmd: NewCmd("pp").
				Opt(NewBoolOpt("bool").Env("BOOL").Default("true")).
				Opt(NewIntOpt("int").Env("INT").Default("123")).
				Opt(NewUintOpt("uint").Env("UINT").Default("456")).
				Opt(NewFloat32Opt("f32").Env("F32").Default("1.23")).
				Opt(NewFloat64Opt("f64").Env("F64").Default("4.56")),
			variations: []testInputOutput{
				{
					// no input, just relying on the default values
					Case: ttCase(),
					args: []string{},
					expected: Command{
						Inputs: []Input{
							{ID: "bool", From: ParsedFrom{Default: true}, RawValue: "true", Value: true},
							{ID: "int", From: ParsedFrom{Default: true}, RawValue: "123", Value: int(123)},
							{ID: "uint", From: ParsedFrom{Default: true}, RawValue: "456", Value: uint(456)},
							{ID: "f32", From: ParsedFrom{Default: true}, RawValue: "1.23", Value: float32(1.23)},
							{ID: "f64", From: ParsedFrom{Default: true}, RawValue: "4.56", Value: float64(4.56)},
						},
					},
				}, {
					// input from args on top for every option
					Case: ttCase(),
					args: []string{
						"--f32", "1.2", "--f64=4.5",
						"--int=12", "--uint", "45",
						"--bool",
					},
					expected: Command{
						Inputs: []Input{
							{ID: "bool", From: ParsedFrom{Default: true}, RawValue: "true", Value: true},
							{ID: "int", From: ParsedFrom{Default: true}, RawValue: "123", Value: int(123)},
							{ID: "uint", From: ParsedFrom{Default: true}, RawValue: "456", Value: uint(456)},
							{ID: "f32", From: ParsedFrom{Default: true}, RawValue: "1.23", Value: float32(1.23)},
							{ID: "f64", From: ParsedFrom{Default: true}, RawValue: "4.56", Value: float64(4.56)},
							{ID: "f32", From: ParsedFrom{Opt: "f32"}, RawValue: "1.2", Value: float32(1.2)},
							{ID: "f64", From: ParsedFrom{Opt: "f64"}, RawValue: "4.5", Value: float64(4.5)},
							{ID: "int", From: ParsedFrom{Opt: "int"}, RawValue: "12", Value: int(12)},
							{ID: "uint", From: ParsedFrom{Opt: "uint"}, RawValue: "45", Value: uint(45)},
							{ID: "bool", From: ParsedFrom{Opt: "bool"}, RawValue: "", Value: true},
						},
					},
				}, {
					// input from both some args and some env vars
					Case: ttCase(),
					envs: map[string]string{
						"F32":  "1.2",
						"UINT": "45",
						"BOOL": "false",
					},
					args: []string{"--f32", "7.89"},
					expected: Command{
						Inputs: []Input{
							{ID: "bool", From: ParsedFrom{Default: true}, RawValue: "true", Value: true},
							{ID: "int", From: ParsedFrom{Default: true}, RawValue: "123", Value: int(123)},
							{ID: "uint", From: ParsedFrom{Default: true}, RawValue: "456", Value: uint(456)},
							{ID: "f32", From: ParsedFrom{Default: true}, RawValue: "1.23", Value: float32(1.23)},
							{ID: "f64", From: ParsedFrom{Default: true}, RawValue: "4.56", Value: float64(4.56)},
							{ID: "bool", From: ParsedFrom{Env: "BOOL"}, RawValue: "false", Value: false},
							{ID: "uint", From: ParsedFrom{Env: "UINT"}, RawValue: "45", Value: uint(45)},
							{ID: "f32", From: ParsedFrom{Env: "F32"}, RawValue: "1.2", Value: float32(1.2)},
							{ID: "f32", From: ParsedFrom{Opt: "f32"}, RawValue: "7.89", Value: float32(7.89)},
						},
					},
				},
			},
		}, {
			// positional arg stuff
			// all required args but not all optional ones
			// missing required args error
			// args with default values
			// surplus
			name: "posargs",
			cmd: NewCmd("posargs").
				Arg(NewArg("arg1").Required()).
				Arg(NewArg("arg2").Required().Env("ARG2")).
				Arg(NewArg("arg3")).
				Arg(NewArg("arg4").Default("Z").Env("ARG4")),
			variations: []testInputOutput{
				{
					Case: ttCase(),
					args: []string{"A", "B", "C", "D", "E", "F"},
					expected: Command{
						Inputs: []Input{
							{ID: "arg4", From: ParsedFrom{Default: true}, RawValue: "Z", Value: "Z"},
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "A", Value: "A"},
							{ID: "arg2", From: ParsedFrom{Arg: 2}, RawValue: "B", Value: "B"},
							{ID: "arg3", From: ParsedFrom{Arg: 3}, RawValue: "C", Value: "C"},
							{ID: "arg4", From: ParsedFrom{Arg: 4}, RawValue: "D", Value: "D"},
						},
						Surplus: []string{"E", "F"},
					},
				}, {
					Case: ttCase(),
					args: []string{"A", "B"},
					expected: Command{
						Inputs: []Input{
							{ID: "arg4", From: ParsedFrom{Default: true}, RawValue: "Z", Value: "Z"},
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "A", Value: "A"},
							{ID: "arg2", From: ParsedFrom{Arg: 2}, RawValue: "B", Value: "B"},
						},
					},
				}, {
					Case: ttCase(),
					envs: map[string]string{"ARG2": "B", "ARG4": "D"},
					args: []string{"A"},
					expected: Command{
						Inputs: []Input{
							{ID: "arg4", From: ParsedFrom{Default: true}, RawValue: "Z", Value: "Z"},
							{ID: "arg2", From: ParsedFrom{Env: "ARG2"}, RawValue: "B", Value: "B"},
							{ID: "arg4", From: ParsedFrom{Env: "ARG4"}, RawValue: "D", Value: "D"},
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "A", Value: "A"},
						},
					},
				}, {
					Case: ttCase(),
					envs: map[string]string{"ARG2": "B"},
					args: []string{"A"},
					expected: Command{
						Inputs: []Input{
							{ID: "arg4", From: ParsedFrom{Default: true}, RawValue: "Z", Value: "Z"},
							{ID: "arg2", From: ParsedFrom{Env: "ARG2"}, RawValue: "B", Value: "B"},
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "A", Value: "A"},
						},
					},
				}, {
					Case:   ttCase(),
					args:   []string{},
					expErr: MissingArgsError{Names: []string{"arg1", "arg2"}},
				}, {
					Case:   ttCase(),
					args:   []string{"A"},
					expErr: MissingArgsError{Names: []string{"arg2"}},
				},
			},
		}, {
			// '--' with '--posarg' after it
			// '=' on an option with and without content
			// mix of `--opt val`, `--opt=val`, and short names
			name: "dashdash_and_eq",
			cmd: NewCmd("ddeq").
				Opt(NewOpt("opt1").Short('o')).
				Arg(NewArg("arg1")),
			variations: []testInputOutput{
				{
					Case: ttCase(),
					args: []string{"--opt1=", "arg1-val"},
					expected: Command{
						Inputs: []Input{
							{ID: "opt1", From: ParsedFrom{Opt: "opt1"}, RawValue: "", Value: ""},
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "arg1-val", Value: "arg1-val"},
						},
					},
				}, {
					Case: ttCase(),
					args: []string{"--", "--opt1="},
					expected: Command{
						Inputs: []Input{
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "--opt1=", Value: "--opt1="},
						},
					},
				}, {
					Case: ttCase(),
					args: []string{"-o=4", "--", "-rf"},
					expected: Command{
						Inputs: []Input{
							{ID: "opt1", From: ParsedFrom{Opt: "o"}, RawValue: "4", Value: "4"},
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "-rf", Value: "-rf"},
						},
					},
				}, {
					Case:     ttCase(),
					args:     []string{"--"},
					expected: Command{},
				}, {
					Case: ttCase(),
					args: []string{"--", "v1", "s1", "s2"},
					expected: Command{
						Inputs: []Input{
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "v1", Value: "v1"},
						},
						Surplus: []string{"s1", "s2"},
					},
				},
			},
		}, {
			// ensure '-' can be a positional argument
			name: "hyphensc",
			cmd: NewCmd("cmd").
				Opt(NewOpt("a")).
				Arg(NewArg("arg1")),
			variations: []testInputOutput{
				{
					Case: ttCase(),
					args: []string{"-aA", "-", "-bB"},
					expected: Command{
						Inputs: []Input{
							{ID: "a", From: ParsedFrom{Opt: "a"}, RawValue: "A", Value: "A"},
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "-", Value: "-"},
						},
						Surplus: []string{"-bB"},
					},
				},
			},
		}, {
			// ensure '-' can be a subcommand
			name: "hyphensc",
			cmd: NewCmd("cmd").
				Opt(NewOpt("a")).
				Subcmd(NewCmd("-").
					Opt(NewOpt("b"))),
			variations: []testInputOutput{
				{
					Case: ttCase(),
					args: []string{"-aA", "-", "-bB"},
					expected: Command{
						Inputs: []Input{
							{ID: "a", From: ParsedFrom{Opt: "a"}, RawValue: "A", Value: "A"},
						},
						Subcmd: &Command{
							Name: "-",
							Inputs: []Input{
								{ID: "b", From: ParsedFrom{Opt: "b"}, RawValue: "B", Value: "B"},
							},
						},
					},
				},
			},
		}, {
			// subcommands (with missing or unknown error checks)
			name: "subcommands",
			cmd: NewCmd("cmd").
				Opt(NewBoolOpt("aa")).
				Subcmd(NewCmd("one").
					Opt(NewOpt("bb")).
					Opt(NewOpt("cc"))).
				Subcmd(NewCmd("two").
					Opt(NewOpt("dd")).
					Opt(NewOpt("ee"))),
			variations: []testInputOutput{
				{
					Case: ttCase(),
					args: []string{"one", "--bb", "B"},
					expected: Command{
						Subcmd: &Command{
							Name: "one",
							Inputs: []Input{
								{ID: "bb", From: ParsedFrom{Opt: "bb"}, RawValue: "B", Value: "B"},
							},
						},
					},
				}, {
					Case: ttCase(),
					args: []string{"two", "--dd", "D"},
					expected: Command{
						Subcmd: &Command{
							Name: "two",
							Inputs: []Input{
								{ID: "dd", From: ParsedFrom{Opt: "dd"}, RawValue: "D", Value: "D"},
							},
						},
					},
				}, {
					Case:   ttCase(),
					args:   []string{"three", "--dd", "D"},
					expErr: UnknownSubcmdError{Name: "three"},
				}, {
					Case:   ttCase(),
					args:   []string{"--aa"},
					expErr: ErrNoSubcmd,
				},
			},
		}, {
			// subcommand help won't require required values
			name: "subcommands",
			cmd: NewCmd("cmd").
				Opt(NewOpt("aa").Required()).
				Subcmd(NewCmd("one").
					Opt(NewOpt("cc").Required())),
			variations: []testInputOutput{
				{
					Case: ttCase(),
					args: []string{"one", "-h"},
					expErr: HelpOrVersionRequested{
						Msg: "cmd one\n\nusage:\n  one [options]\n\noptions:\n      --cc  <arg>   (required)\n  -h, --help        Show this help message and exit.\n",
					},
				}, {
					Case:   ttCase(),
					args:   []string{"--aa=1", "one"},
					expErr: MissingOptionsError{Names: []string{"--cc"}},
				},
			},
		}, {
			// custom parser
			name: "custom_parser",
			cmd: NewCmd("cp").
				Opt(NewOpt("aa").
					WithParser(func(s string) (any, error) {
						comma := strings.IndexByte(s, ',')
						x, _ := strconv.Atoi(s[:comma])
						y, _ := strconv.Atoi(s[comma+1:])
						return image.Point{X: x, Y: y}, nil
					})),
			variations: []testInputOutput{
				{
					Case: ttCase(),
					args: []string{"--aa", "3,7"},
					expected: Command{
						Inputs: []Input{
							{ID: "aa", From: ParsedFrom{Opt: "aa"}, RawValue: "3,7", Value: image.Point{3, 7}},
						},
					},
				},
			},
		}, {
			// stacking / bunching short options and their values
			name: "shortstacks",
			cmd: NewCmd("shst").
				Opt(NewBoolOpt("bb").Short('b')).
				Opt(NewOpt("aa").Short('a')).
				Opt(NewBoolOpt("cc").Short('c')),
			variations: []testInputOutput{
				{
					Case: ttCase(),
					args: []string{"-bc"},
					expected: Command{
						Inputs: []Input{
							{ID: "bb", From: ParsedFrom{Opt: "b"}, RawValue: "", Value: true},
							{ID: "cc", From: ParsedFrom{Opt: "c"}, RawValue: "", Value: true},
						},
					},
				}, {
					Case: ttCase(),
					args: []string{"-bbb"},
					expected: Command{
						Inputs: []Input{
							{ID: "bb", From: ParsedFrom{Opt: "b"}, RawValue: "", Value: true},
							{ID: "bb", From: ParsedFrom{Opt: "b"}, RawValue: "", Value: true},
							{ID: "bb", From: ParsedFrom{Opt: "b"}, RawValue: "", Value: true},
						},
					},
				}, {
					Case: ttCase(),
					args: []string{"-cb"},
					expected: Command{
						Inputs: []Input{
							{ID: "cc", From: ParsedFrom{Opt: "c"}, RawValue: "", Value: true},
							{ID: "bb", From: ParsedFrom{Opt: "b"}, RawValue: "", Value: true},
						},
					},
				}, {
					Case:   ttCase(),
					args:   []string{"-cba"},
					expErr: MissingOptionValueError{Name: "a"},
				}, {
					Case: ttCase(),
					args: []string{"-cb", "-a", "valA"},
					expected: Command{
						Inputs: []Input{
							{ID: "cc", From: ParsedFrom{Opt: "c"}, RawValue: "", Value: true},
							{ID: "bb", From: ParsedFrom{Opt: "b"}, RawValue: "", Value: true},
							{ID: "aa", From: ParsedFrom{Opt: "a"}, RawValue: "valA", Value: "valA"},
						},
					},
				}, {
					Case: ttCase(),
					args: []string{"-cba", "valA"},
					expected: Command{
						Inputs: []Input{
							{ID: "cc", From: ParsedFrom{Opt: "c"}, RawValue: "", Value: true},
							{ID: "bb", From: ParsedFrom{Opt: "b"}, RawValue: "", Value: true},
							{ID: "aa", From: ParsedFrom{Opt: "a"}, RawValue: "valA", Value: "valA"},
						},
					},
				}, {
					Case: ttCase(),
					args: []string{"-cab"},
					expected: Command{
						Inputs: []Input{
							{ID: "cc", From: ParsedFrom{Opt: "c"}, RawValue: "", Value: true},
							{ID: "aa", From: ParsedFrom{Opt: "a"}, RawValue: "b", Value: "b"},
						},
					},
				}, {
					Case: ttCase(),
					args: []string{"--a", "v"},
					expected: Command{
						Inputs: []Input{
							{ID: "aa", From: ParsedFrom{Opt: "a"}, RawValue: "v", Value: "v"},
						},
					},
				}, {
					Case:   ttCase(),
					args:   []string{"-bz"},
					expErr: UnknownOptionError{Name: "-z"},
				}, {
					Case: ttCase(),
					args: []string{"-aa", "v"},
					expected: Command{
						Inputs: []Input{
							{ID: "aa", From: ParsedFrom{Opt: "a"}, RawValue: "a", Value: "a"},
						},
						Surplus: []string{"v"},
					},
				},
			},
		}, {
			// versioning (on just the top level for now)
			// using the builder method for a custom versioner
			name: "versioning",
			cmd: NewCmd("cmd").
				Opt(NewOpt("a")).
				Opt(NewBoolOpt("b")).
				Opt(NewBoolOpt("v").WithVersioner(func(_ Input) string { return "version-string" })),
			variations: []testInputOutput{
				{
					Case: ttCase(),
					args: []string{"-v"},
					expErr: HelpOrVersionRequested{
						Msg: "version-string",
					},
				}, {
					Case: ttCase(),
					args: []string{"-a", "A"},
					expected: Command{
						Inputs: []Input{
							{ID: "a", From: ParsedFrom{Opt: "a"}, RawValue: "A", Value: "A"},
						},
					},
				}, {
					Case: ttCase(),
					args: []string{"-bv"},
					expErr: HelpOrVersionRequested{
						Msg: "version-string",
					},
				},
			},
		}, {
			// versioning (on just the top level for now)
			// using the constructor for a version option
			name: "versioning",
			cmd:  NewCmd("cmd").Opt(DefaultVersionOpt),
			variations: []testInputOutput{
				{
					Case:   ttCase(),
					args:   []string{"--version"},
					expErr: HelpOrVersionRequested{Msg: "(devel)\n"},
				}, {
					Case:   ttCase(),
					args:   []string{"-v"},
					expErr: HelpOrVersionRequested{Msg: "(devel)\n"},
				},
			},
		}, {
			// versioning (on just the top level for now)
			// using the constructor for a version option but leaving out a short name
			name: "versioning",
			cmd:  NewCmd("cmd").Opt(NewVersionOpt(0, "version", VersionOptConfig{})),
			variations: []testInputOutput{
				{
					Case:   ttCase(),
					args:   []string{"--version"},
					expErr: HelpOrVersionRequested{Msg: "(devel)\n"},
				},
				{Case: ttCase(), args: []string{"-v"}, expErr: UnknownOptionError{Name: "-v"}},
				{Case: ttCase(), args: []string{"-V"}, expErr: UnknownOptionError{Name: "-V"}},
			},
		}, {
			// versioning (on just the top level for now)
			// using the constructor for a version option but leaving out a long name
			name: "versioning",
			cmd:  NewCmd("cmd").Opt(NewVersionOpt('Z', "", VersionOptConfig{})),
			variations: []testInputOutput{
				{
					Case:   ttCase(),
					args:   []string{"-Z"},
					expErr: HelpOrVersionRequested{Msg: "(devel)\n"},
				}, {
					Case:   ttCase(),
					args:   []string{"--version"},
					expErr: UnknownOptionError{Name: "--version"},
				},
			},
		},
	} {
		for tioIdx, tio := range tt.variations {
			t.Run(fmt.Sprintf("%s %d", tt.name, tioIdx), func(t *testing.T) {
				for k, v := range tio.envs {
					t.Setenv(k, v)
				}

				got, gotErr := tt.cmd.ParseThese(tio.args...)
				if tio.expErr != nil && gotErr == nil {
					t.Fatalf("expected error %[1]T: %[1]v, got no error", tio.expErr)
				}
				if gotErr != nil {
					if tio.expErr == nil {
						t.Fatalf("expected no error, got %[1]T: %[1]v", gotErr)
					}
					if !errors.Is(gotErr, tio.expErr) {
						t.Fatalf("%s: errors don't match:\nexpected: (%[2]T) %+#[2]v\n     got: (%[3]T) %+#[3]v",
							tio.Case, tio.expErr, gotErr)
					}
					return
				}

				cmpParsed(t, tio.Case, &tio.expected, got)
			})
		}
	}
}

func cmpParsed(t *testing.T, tioInfo string, exp, got *Command) {
	t.Helper()

	// command name
	if got.Name != exp.Name {
		t.Errorf("%s:\nexpected command name '%s', got '%s'", tioInfo, exp.Name, got.Name)
	}
	// inputs
	{
		gotNumInputs := len(got.Inputs)
		expNumInputs := len(exp.Inputs)
		if gotNumInputs != expNumInputs {
			t.Fatalf("%s: expected %d parsed inputs, got %d", tioInfo, expNumInputs, gotNumInputs)
		}
		for i, gotOpt := range got.Inputs {
			expOpt := exp.Inputs[i]
			if !reflect.DeepEqual(gotOpt, expOpt) {
				t.Errorf("%s: parsed inputs[%d]:\nexpected %+#v\n     got %+#v", tioInfo, i, expOpt, gotOpt)
			}
		}
	}
	// surplus args
	{
		if !slices.Equal(got.Surplus, exp.Surplus) {
			t.Errorf("%s: surplus args:\nexpected %+#v\n     got %+#v",
				tioInfo, exp.Surplus, got.Surplus)
		}
	}
	// subcommand
	{
		switch {
		case got.Subcmd == nil && exp.Subcmd != nil:
			t.Errorf("%s:\nexpected subcommand %+v\ngot nil", tioInfo, exp.Subcmd)
		case got.Subcmd != nil && exp.Subcmd == nil:
			t.Errorf("%s:\ndid not expect a subcommand\ngot %+v", tioInfo, got.Subcmd)
		case got.Subcmd != nil && exp.Subcmd != nil:
			cmpParsed(t, tioInfo, exp.Subcmd, got.Subcmd)
		}
	}
}

func TestCmpErrors(t *testing.T) {
	for ttIdx, tt := range []struct {
		err      error
		target   error
		expected bool
	}{
		{
			err:      MissingOptionsError{Names: []string{"-a", "--bb"}},
			target:   MissingOptionsError{Names: []string{"-a", "--bb"}},
			expected: true,
		}, {
			err:      MissingOptionsError{Names: []string{"-a", "--bb"}},
			target:   MissingOptionsError{Names: []string{"-c"}},
			expected: false,
		}, {
			err:      MissingArgsError{Names: []string{"a", "b"}},
			target:   MissingArgsError{Names: []string{"a", "b"}},
			expected: true,
		}, {
			err:      MissingArgsError{Names: []string{"a", "b"}},
			target:   MissingArgsError{Names: []string{"c"}},
			expected: false,
		}, {
			err:      UnknownSubcmdError{Name: "a"},
			target:   UnknownSubcmdError{Name: "a"},
			expected: true,
		}, {
			err:      UnknownSubcmdError{Name: "c"},
			target:   UnknownSubcmdError{Name: "d"},
			expected: false,
		}, {
			err:      ErrNoSubcmd,
			target:   ErrNoSubcmd,
			expected: true,
		},
	} {
		if errors.Is(tt.err, tt.target) != tt.expected {
			t.Fatalf("tt[%d]: base", ttIdx)
		}
		err := fmt.Errorf("wrapped: %w", tt.err)
		if errors.Is(err, tt.target) != tt.expected {
			t.Fatalf("tt[%d]: wrapped", ttIdx)
		}
		err = fmt.Errorf("wrapped again: %w", err)
		if errors.Is(err, tt.target) != tt.expected {
			t.Fatalf("tt[%d]: wrapped again", ttIdx)
		}
	}
}

func TestOptLookups(t *testing.T) {
	in := NewCmd("program").
		Opt(NewOpt("a").Required()).
		Opt(NewOpt("b"))

	// with both options present
	{
		c := in.ParseTheseOrExit("-ahello", "-bworld")
		// straight getting the opts
		{
			optA := Get[string](c, "a")
			if optA != "hello" {
				t.Errorf(`expected "hello" for option a, got "%s"`, optA)
			}
			optB := Get[string](c, "b")
			if optB != "world" {
				t.Errorf(`expected "world" for option b, got "%s"`, optB)
			}
		}
		// above should be identical with lookups
		{
			optA, ok := Lookup[string](c, "a")
			if optA != "hello" || !ok {
				t.Errorf(`expected ("hello", true) for option a lookup, got ("%s", %v)`, optA, ok)
			}
			optB, ok := Lookup[string](c, "b")
			if optB != "world" || !ok {
				t.Errorf(`expected ("world", true) for option b lookup, got ("%s", %v)`, optB, ok)
			}
		}
	}

	// with only the first option 'a' present
	{
		c := in.ParseTheseOrExit("-ahello")
		// first one should be there, second one shouldn't
		{
			optA, ok := Lookup[string](c, "a")
			if optA != "hello" || !ok {
				t.Errorf(`expected ("hello", true) for option 'a' lookup, got ("%s", %v)`, optA, ok)
			}
			optB, ok := Lookup[string](c, "b")
			if optB != "" || ok {
				t.Errorf(`expected ("", false) for option 'b' lookup, got (%s, %v)`, optB, ok)
			}
		}
		// based on the above assertions, we should get the right fallback for option 'b'
		{
			optA := GetOr(c, "a", "hey")
			if optA != "hello" {
				t.Errorf(`expected "hello" for option 'a', got "%s"`, optA)
			}
			optB := GetOr(c, "b", "earth")
			if optB != "earth" {
				t.Errorf(`expected "earth" for option 'b' fallback, got "%s"`, optB)
			}
		}
		// trying to straight get the second one should panic
		{
			func() {
				defer func() {
					gotPanicVal := recover()
					expPanicVal := "no parsed input value for id 'b'"
					if !reflect.DeepEqual(gotPanicVal, expPanicVal) {
						t.Fatalf("panic values don't match\nexpected: %+#v\n     got: %+#v",
							expPanicVal, gotPanicVal)
					}
				}()
				_ = Get[string](c, "b")
			}()
		}
		// trying to cast the first one to anything but a string should panic
		{
			func() {
				defer func() {
					gotPanicVal := recover()
					if gotPanicVal == nil {
						t.Fatalf("didn't panic when trying to cast a string arg to an int")
					}
				}()
				_ = Get[int](c, "a")
			}()
		}
	}

	in = NewCmd("program").
		Opt(NewOpt("a").Default("A")).
		Opt(NewOpt("b").Default("B"))

	// with both options present
	{
		c := in.ParseTheseOrExit("-ahello", "-bworld")
		// straight getting the opts
		{
			optA := Get[string](c, "a")
			if optA != "hello" {
				t.Errorf(`expected "hello" for option a, got "%s"`, optA)
			}
			optB := Get[string](c, "b")
			if optB != "world" {
				t.Errorf(`expected "world" for option b, got "%s"`, optB)
			}
		}
		// above should be identical with lookups
		{
			optA, ok := Lookup[string](c, "a")
			if optA != "hello" || !ok {
				t.Errorf(`expected ("hello", true) for option a lookup, got ("%s", %v)`, optA, ok)
			}
			optB, ok := Lookup[string](c, "b")
			if optB != "world" || !ok {
				t.Errorf(`expected ("world", true) for option b lookup, got ("%s", %v)`, optB, ok)
			}
		}
	}

	// with only the first option 'a' provided
	{
		c := in.ParseTheseOrExit("-ahello")
		// first one should be there, second one default
		{
			optA, ok := Lookup[string](c, "a")
			if optA != "hello" || !ok {
				t.Errorf(`expected ("hello", true) for option 'a' lookup, got ("%s", %v)`, optA, ok)
			}
			optB, ok := Lookup[string](c, "b")
			if optB != "B" || !ok {
				t.Errorf(`expected ("B", true) for option 'b' lookup, got (%s, %v)`, optB, ok)
			}
		}
		// based on the above assertions, we should get the right fallback for option 'b'
		{
			optA := GetOr(c, "a", "hey")
			if optA != "hello" {
				t.Errorf(`expected "hello" for option 'a', got "%s"`, optA)
			}
			optB := GetOr(c, "b", "earth")
			if optB != "B" {
				t.Errorf(`expected to not use fallback for option 'b' and get "B", got "%s"`, optB)
			}
		}
		// trying to cast the first one to anything but a string should panic
		{
			func() {
				defer func() {
					gotPanicVal := recover()
					if gotPanicVal == nil {
						t.Fatalf("didn't panic when trying to cast a string arg to an int")
					}
				}()
				_ = Get[int](c, "a")
			}()
		}
	}
}

func TestArgLookups(t *testing.T) {
	in := NewCmd("program").
		Arg(NewArg("arg1").Required()).
		Arg(NewArg("arg2"))

	// with both args present
	{
		c := in.ParseTheseOrExit("hello", "world")
		// straight getting the args
		{
			arg1 := Get[string](c, "arg1")
			if arg1 != "hello" {
				t.Errorf(`expected "hello" for arg1, got "%s"`, arg1)
			}
			arg2 := Get[string](c, "arg2")
			if arg2 != "world" {
				t.Errorf(`expected "world" for arg2, got "%s"`, arg2)
			}
		}
		// above should be identical with lookups
		{
			arg1, ok := Lookup[string](c, "arg1")
			if arg1 != "hello" || !ok {
				t.Errorf(`expected ("hello", true) for arg1 lookup, got ("%s", %v)`, arg1, ok)
			}
			arg2, ok := Lookup[string](c, "arg2")
			if arg2 != "world" || !ok {
				t.Errorf(`expected ("world", true) for arg2, got ("%s", %v)`, arg2, ok)
			}
		}
	}

	// with only the first arg 'arg1' present
	{
		c := in.ParseTheseOrExit("hello")
		// first one should be there, second one shouldn't
		{
			arg1, ok := Lookup[string](c, "arg1")
			if arg1 != "hello" || !ok {
				t.Errorf(`expected ("hello", true) for arg1 lookup, got ("%s", %v)`, arg1, ok)
			}
			arg2, ok := Lookup[string](c, "arg2")
			if arg2 != "" || ok {
				t.Errorf(`expected ("", false) for arg2, got (%s, %v)`, arg2, ok)
			}
		}
		// based on the above assertions, we should get the right fallback for arg2
		{
			arg1 := GetOr(c, "arg1", "hey")
			if arg1 != "hello" {
				t.Errorf(`expected "hello" for arg1, got "%s"`, arg1)
			}
			arg2 := GetOr(c, "arg2", "earth")
			if arg2 != "earth" {
				t.Errorf(`expected "earth" for arg2, got "%s"`, arg2)
			}
		}
		// trying to straight get the second one should panic
		{
			func() {
				defer func() {
					gotPanicVal := recover()
					expPanicVal := "no parsed input value for id 'arg2'"
					if !reflect.DeepEqual(gotPanicVal, expPanicVal) {
						t.Fatalf("panic values don't match\nexpected: %+#v\n     got: %+#v",
							expPanicVal, gotPanicVal)
					}
				}()
				_ = Get[string](c, "arg2")
			}()
		}
		// trying to cast the first one to anything but a string should panic
		{
			func() {
				defer func() {
					gotPanicVal := recover()
					if gotPanicVal == nil {
						t.Fatalf("didn't panic when trying to cast a string arg to an int")
					}
				}()
				_ = Get[int](c, "arg1")
			}()
		}
	}
}

func TestHelpSubcommands(t *testing.T) {
	for _, tt := range []struct {
		Case       string
		cmd        CommandInfo
		cliArgs    []string
		expHelpMsg string
	}{
		{
			Case: ttCase(),
			cmd: NewCmd("example").
				Help("nested two levels").
				Subcmd(NewCmd("a").
					Subcmd(NewCmd("b").
						Help("subcommand b"))),
			cliArgs: []string{"a", "b", "-h"},
			expHelpMsg: `example a b - subcommand b

usage:
  b [options]

options:
  -h, --help   Show this help message and exit.
`,
		}, {
			Case: ttCase(),
			cmd: NewCmd("example").
				Help("nested three levels").
				Subcmd(NewCmd("a").
					Subcmd(NewCmd("b").
						Subcmd(NewCmd("c").
							Help("subcommand c")))),
			cliArgs: []string{"a", "b", "c", "-h"},
			expHelpMsg: `example a b c - subcommand c

usage:
  c [options]

options:
  -h, --help   Show this help message and exit.
`,
		},
	} {
		_, err := tt.cmd.ParseThese(tt.cliArgs...)
		gotHelpMsg := err.Error()
		if gotHelpMsg != tt.expHelpMsg {
			t.Errorf("%s: expected:\n%s\ngot:\n%s", tt.Case, tt.expHelpMsg, gotHelpMsg)
		}
	}
}

func ttCase() string {
	_, _, line, _ := runtime.Caller(1)
	return fmt.Sprintf("tt:%d", line)
}
