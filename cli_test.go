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
		ttInfo   string
		envs     map[string]string
		args     []string
		expected Command
		expErr   error
	}
	type testCase struct {
		name       string
		cmd        *RootCommandInfo
		variations []testInputOutput
	}

	for _, tt := range []testCase{
		// no positional args or subcommands
		// errors for missing required opts
		{
			name: "options_only",
			cmd: NewCmd("optsonly").
				Opt(NewBoolOpt("aa")).
				Opt(NewOpt("bb").ShortOnly('b')).
				Opt(NewOpt("cc").Required()).
				Opt(NewOpt("dd").Default("v4")).
				Opt(NewOpt("ee")).
				Build(),
			variations: []testInputOutput{
				{
					ttInfo: ttCase(),
					args:   []string{"-b", "v2", "--aa", "--cc=v3"},
					expected: Command{
						Opts: []Input{
							{ID: "dd", From: ParsedFrom{RawDefault: true}, RawValue: "v4", Value: "v4"},
							{ID: "bb", From: ParsedFrom{Opt: "b"}, RawValue: "v2", Value: "v2"},
							{ID: "aa", From: ParsedFrom{Opt: "aa"}, RawValue: "", Value: true},
							{ID: "cc", From: ParsedFrom{Opt: "cc"}, RawValue: "v3", Value: "v3"},
						},
					},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"-b", "v2", "--aa"},
					expErr: MissingOptionsError{Names: []string{"--cc"}},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"-z"},
					expErr: UnknownOptionError{Name: "-z"},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"--zz=abc"},
					expErr: UnknownOptionError{Name: "--zz=abc"},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"--bb", "B"},
					expErr: UnknownOptionError{Name: "--bb"},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"--dd"},
					expErr: MissingOptionValueError{Name: "dd"},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"-b"},
					expErr: MissingOptionValueError{Name: "b"},
				},
			},
		},
		// all provided parsers with defaults
		{
			name: "provided_parsers",
			cmd: NewCmd("pp").
				Opt(NewBoolOpt("bool").Env("BOOL").Default("true")).
				Opt(NewIntOpt("int").Env("INT").Default("123")).
				Opt(NewUintOpt("uint").Env("UINT").Default("456")).
				Opt(NewFloat32Opt("f32").Env("F32").Default("1.23")).
				Opt(NewFloat64Opt("f64").Env("F64").Default("4.56")).
				Build(),
			variations: []testInputOutput{
				// no input, just relying on the default values
				{
					ttInfo: ttCase(),
					args:   []string{},
					expected: Command{
						Opts: []Input{
							{ID: "bool", From: ParsedFrom{RawDefault: true}, RawValue: "true", Value: true},
							{ID: "int", From: ParsedFrom{RawDefault: true}, RawValue: "123", Value: int(123)},
							{ID: "uint", From: ParsedFrom{RawDefault: true}, RawValue: "456", Value: uint(456)},
							{ID: "f32", From: ParsedFrom{RawDefault: true}, RawValue: "1.23", Value: float32(1.23)},
							{ID: "f64", From: ParsedFrom{RawDefault: true}, RawValue: "4.56", Value: float64(4.56)},
						},
					},
				},
				// input from args on top for every option
				{
					ttInfo: ttCase(),
					args: []string{
						"--f32", "1.2", "--f64=4.5",
						"--int=12", "--uint", "45",
						"--bool",
					},
					expected: Command{
						Opts: []Input{
							{ID: "bool", From: ParsedFrom{RawDefault: true}, RawValue: "true", Value: true},
							{ID: "int", From: ParsedFrom{RawDefault: true}, RawValue: "123", Value: int(123)},
							{ID: "uint", From: ParsedFrom{RawDefault: true}, RawValue: "456", Value: uint(456)},
							{ID: "f32", From: ParsedFrom{RawDefault: true}, RawValue: "1.23", Value: float32(1.23)},
							{ID: "f64", From: ParsedFrom{RawDefault: true}, RawValue: "4.56", Value: float64(4.56)},
							{ID: "f32", From: ParsedFrom{Opt: "f32"}, RawValue: "1.2", Value: float32(1.2)},
							{ID: "f64", From: ParsedFrom{Opt: "f64"}, RawValue: "4.5", Value: float64(4.5)},
							{ID: "int", From: ParsedFrom{Opt: "int"}, RawValue: "12", Value: int(12)},
							{ID: "uint", From: ParsedFrom{Opt: "uint"}, RawValue: "45", Value: uint(45)},
							{ID: "bool", From: ParsedFrom{Opt: "bool"}, RawValue: "", Value: true},
						},
					},
				},
				// input from both some args and some env vars
				{
					ttInfo: ttCase(),
					envs: map[string]string{
						"F32":  "1.2",
						"UINT": "45",
						"BOOL": "false",
					},
					args: []string{"--f32", "7.89"},
					expected: Command{
						Opts: []Input{
							{ID: "bool", From: ParsedFrom{RawDefault: true}, RawValue: "true", Value: true},
							{ID: "int", From: ParsedFrom{RawDefault: true}, RawValue: "123", Value: int(123)},
							{ID: "uint", From: ParsedFrom{RawDefault: true}, RawValue: "456", Value: uint(456)},
							{ID: "f32", From: ParsedFrom{RawDefault: true}, RawValue: "1.23", Value: float32(1.23)},
							{ID: "f64", From: ParsedFrom{RawDefault: true}, RawValue: "4.56", Value: float64(4.56)},
							{ID: "bool", From: ParsedFrom{Env: "BOOL"}, RawValue: "false", Value: false},
							{ID: "uint", From: ParsedFrom{Env: "UINT"}, RawValue: "45", Value: uint(45)},
							{ID: "f32", From: ParsedFrom{Env: "F32"}, RawValue: "1.2", Value: float32(1.2)},
							{ID: "f32", From: ParsedFrom{Opt: "f32"}, RawValue: "7.89", Value: float32(7.89)},
						},
					},
				},
			},
		},
		// positional arg stuff
		// all required args but not all optional ones
		// missing required args error
		// args with default values
		// surplus
		{
			name: "posargs",
			cmd: NewCmd("posargs").
				Arg(NewArg("arg1").Required()).
				Arg(NewArg("arg2").Required().Env("ARG2")).
				Arg(NewArg("arg3")).
				Arg(NewArg("arg4").Default("Z").Env("ARG4")).
				Build(),
			variations: []testInputOutput{
				{
					ttInfo: ttCase(),
					args:   []string{"A", "B", "C", "D", "E", "F"},
					expected: Command{
						Args: []Input{
							{ID: "arg4", From: ParsedFrom{RawDefault: true}, RawValue: "Z", Value: "Z"},
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "A", Value: "A"},
							{ID: "arg2", From: ParsedFrom{Arg: 2}, RawValue: "B", Value: "B"},
							{ID: "arg3", From: ParsedFrom{Arg: 3}, RawValue: "C", Value: "C"},
							{ID: "arg4", From: ParsedFrom{Arg: 4}, RawValue: "D", Value: "D"},
						},
						Surplus: []string{"E", "F"},
					},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"A", "B"},
					expected: Command{
						Args: []Input{
							{ID: "arg4", From: ParsedFrom{RawDefault: true}, RawValue: "Z", Value: "Z"},
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "A", Value: "A"},
							{ID: "arg2", From: ParsedFrom{Arg: 2}, RawValue: "B", Value: "B"},
						},
					},
				},
				{
					ttInfo: ttCase(),
					envs:   map[string]string{"ARG2": "B", "ARG4": "D"},
					args:   []string{"A"},
					expected: Command{
						Args: []Input{
							{ID: "arg4", From: ParsedFrom{RawDefault: true}, RawValue: "Z", Value: "Z"},
							{ID: "arg2", From: ParsedFrom{Env: "ARG2"}, RawValue: "B", Value: "B"},
							{ID: "arg4", From: ParsedFrom{Env: "ARG4"}, RawValue: "D", Value: "D"},
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "A", Value: "A"},
						},
					},
				},
				{
					ttInfo: ttCase(),
					envs:   map[string]string{"ARG2": "B"},
					args:   []string{"A"},
					expected: Command{
						Args: []Input{
							{ID: "arg4", From: ParsedFrom{RawDefault: true}, RawValue: "Z", Value: "Z"},
							{ID: "arg2", From: ParsedFrom{Env: "ARG2"}, RawValue: "B", Value: "B"},
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "A", Value: "A"},
						},
					},
				},
				{
					ttInfo: ttCase(),
					args:   []string{},
					expErr: MissingArgsError{Names: []string{"arg1", "arg2"}},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"A"},
					expErr: MissingArgsError{Names: []string{"arg2"}},
				},
			},
		},
		// '--' with '--posarg' after it
		// '=' on an option with and without content
		// mix of `--opt val`, `--opt=val`, and short names
		{
			name: "dashdash_and_eq",
			cmd: NewCmd("ddeq").
				Opt(NewOpt("opt1").Short('o')).
				Arg(NewArg("arg1")).
				Build(),
			variations: []testInputOutput{
				{
					ttInfo: ttCase(),
					args:   []string{"--opt1=", "arg1-val"},
					expected: Command{
						Opts: []Input{
							{ID: "opt1", From: ParsedFrom{Opt: "opt1"}, RawValue: "", Value: ""},
						},
						Args: []Input{
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "arg1-val", Value: "arg1-val"},
						},
					},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"--", "--opt1="},
					expected: Command{
						Args: []Input{
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "--opt1=", Value: "--opt1="},
						},
					},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"-o=4", "--", "-rf"},
					expected: Command{
						Opts: []Input{
							{ID: "opt1", From: ParsedFrom{Opt: "o"}, RawValue: "4", Value: "4"},
						},
						Args: []Input{
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "-rf", Value: "-rf"},
						},
					},
				},
				{
					ttInfo:   ttCase(),
					args:     []string{"--"},
					expected: Command{},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"--", "v1", "s1", "s2"},
					expected: Command{
						Args: []Input{
							{ID: "arg1", From: ParsedFrom{Arg: 1}, RawValue: "v1", Value: "v1"},
						},
						Surplus: []string{"s1", "s2"},
					},
				},
			},
		},
		// subcommands (with missing or uknown error checks)
		{
			name: "subcommands",
			cmd: NewCmd("cmd").
				Opt(NewBoolOpt("aa")).
				Subcmd(NewCmd("one").
					Opt(NewOpt("bb")).
					Opt(NewOpt("cc"))).
				Subcmd(NewCmd("two").
					Opt(NewOpt("dd")).
					Opt(NewOpt("ee"))).
				Build(),
			variations: []testInputOutput{
				{
					ttInfo: ttCase(),
					args:   []string{"one", "--bb", "B"},
					expected: Command{
						Subcmd: &Command{
							Opts: []Input{
								{ID: "bb", From: ParsedFrom{Opt: "bb"}, RawValue: "B", Value: "B"},
							},
						},
					},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"two", "--dd", "D"},
					expected: Command{
						Subcmd: &Command{
							Opts: []Input{
								{ID: "dd", From: ParsedFrom{Opt: "dd"}, RawValue: "D", Value: "D"},
							},
						},
					},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"three", "--dd", "D"},
					expErr: UnknownSubcmdError{Name: "three"},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"--aa"},
					expErr: ErrNoSubcmd,
				},
			},
		},
		// subcommand help won't require required values
		{
			name: "subcommands",
			cmd: NewCmd("cmd").
				Opt(NewOpt("aa").Required()).
				Subcmd(NewCmd("one").
					Opt(NewOpt("cc").Required())).
				Build(),
			variations: []testInputOutput{
				{
					ttInfo: ttCase(),
					args:   []string{"one", "-h"},
					expErr: HelpRequestError{
						HelpMsg: "cmd one - \n\nusage:\n  one [options]\n\noptions:\n      --cc  <arg>   (required)\n  -h, --help        Show this help message and exit.\n",
					},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"--aa=1", "one"},
					expErr: MissingOptionsError{Names: []string{"--cc"}},
				},
			},
		},
		// custom parser
		{
			name: "custom_parser",
			cmd: NewCmd("cp").
				Opt(NewOpt("aa").
					WithParser(func(s string) (any, error) {
						comma := strings.IndexByte(s, ',')
						x, _ := strconv.Atoi(s[:comma])
						y, _ := strconv.Atoi(s[comma+1:])
						return image.Point{X: x, Y: y}, nil
					})).
				Build(),
			variations: []testInputOutput{
				{
					ttInfo: ttCase(),
					args:   []string{"--aa", "3,7"},
					expected: Command{
						Opts: []Input{
							{ID: "aa", From: ParsedFrom{Opt: "aa"}, RawValue: "3,7", Value: image.Point{3, 7}},
						},
					},
				},
			},
		},
		// stacking / bunching short options and their values
		{
			name: "shortstacks",
			cmd: NewCmd("shst").
				Opt(NewBoolOpt("bb").Short('b')).
				Opt(NewOpt("aa").Short('a')).
				Opt(NewBoolOpt("cc").Short('c')).
				Build(),
			variations: []testInputOutput{
				{
					ttInfo: ttCase(),
					args:   []string{"-bc"},
					expected: Command{
						Opts: []Input{
							{ID: "bb", From: ParsedFrom{Opt: "b"}, RawValue: "", Value: true},
							{ID: "cc", From: ParsedFrom{Opt: "c"}, RawValue: "", Value: true},
						},
					},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"-cb"},
					expected: Command{
						Opts: []Input{
							{ID: "cc", From: ParsedFrom{Opt: "c"}, RawValue: "", Value: true},
							{ID: "bb", From: ParsedFrom{Opt: "b"}, RawValue: "", Value: true},
						},
					},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"-cba"},
					expErr: MissingOptionValueError{Name: "a"},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"-cb", "-a", "valA"},
					expected: Command{
						Opts: []Input{
							{ID: "cc", From: ParsedFrom{Opt: "c"}, RawValue: "", Value: true},
							{ID: "bb", From: ParsedFrom{Opt: "b"}, RawValue: "", Value: true},
							{ID: "aa", From: ParsedFrom{Opt: "a"}, RawValue: "valA", Value: "valA"},
						},
					},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"-cba", "valA"},
					expected: Command{
						Opts: []Input{
							{ID: "cc", From: ParsedFrom{Opt: "c"}, RawValue: "", Value: true},
							{ID: "bb", From: ParsedFrom{Opt: "b"}, RawValue: "", Value: true},
							{ID: "aa", From: ParsedFrom{Opt: "a"}, RawValue: "valA", Value: "valA"},
						},
					},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"-cab"},
					expected: Command{
						Opts: []Input{
							{ID: "cc", From: ParsedFrom{Opt: "c"}, RawValue: "", Value: true},
							{ID: "aa", From: ParsedFrom{Opt: "a"}, RawValue: "b", Value: "b"},
						},
					},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"--a", "v"},
					expected: Command{
						Opts: []Input{
							{ID: "aa", From: ParsedFrom{Opt: "a"}, RawValue: "v", Value: "v"},
						},
					},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"-bz"},
					expErr: UnknownOptionError{Name: "-z"},
				},
				{
					ttInfo: ttCase(),
					args:   []string{"-aa", "v"},
					expected: Command{
						Opts: []Input{
							{ID: "aa", From: ParsedFrom{Opt: "a"}, RawValue: "a", Value: "a"},
						},
						Surplus: []string{"v"},
					},
				},
			},
		},
	} {
		for tioIdx, tio := range tt.variations {
			t.Run(fmt.Sprintf("%s %d", tt.name, tioIdx), func(t *testing.T) {
				for k, v := range tio.envs {
					t.Setenv(k, v)
				}

				got, gotErr := tt.cmd.Parse(tio.args...)
				if tio.expErr != nil && gotErr == nil {
					t.Fatalf("expected error %[1]T: %[1]v, got no error", tio.expErr)
				}
				if gotErr != nil {
					if tio.expErr == nil {
						t.Fatalf("expected no error, got %[1]T: %[1]v", gotErr)
					}
					if !errors.Is(gotErr, tio.expErr) {
						t.Fatalf("tt:%s: errors don't match:\nexpected: (%[2]T) %+#[2]v\n     got: (%[3]T) %+#[3]v",
							tio.ttInfo, tio.expErr, gotErr)
					}
					return
				}

				cmpParsed(t, tio.ttInfo, tio.expected, got)
			})
		}
	}
}

func cmpParsed(t *testing.T, tioInfo string, exp, got Command) {
	t.Helper()

	// options
	{
		expNumOpts := len(exp.Opts)
		gotNumOpts := len(got.Opts)
		if gotNumOpts != expNumOpts {
			t.Fatalf("tt:%s: expected %d parsed options, got %d", tioInfo, expNumOpts, gotNumOpts)
		}
		for i, gotOpt := range got.Opts {
			expOpt := exp.Opts[i]
			if !reflect.DeepEqual(gotOpt, expOpt) {
				t.Errorf("tt:%s: parsed options[%d]:\nexpected %+#v\n     got %+#v", tioInfo, i, expOpt, gotOpt)
			}
		}
	}
	// positional arguments
	{
		expNumArgs := len(exp.Args)
		gotNumArgs := len(got.Args)
		if gotNumArgs != expNumArgs {
			t.Fatalf("tt:%s: expected %d parsed positional arguments, got %d", tioInfo, expNumArgs, gotNumArgs)
		}
		for i, gotArg := range got.Args {
			expArg := exp.Args[i]
			if !reflect.DeepEqual(gotArg, expArg) {
				t.Errorf("tt:%s: parsed options[%d]:\nexpected %+#v\n     got %+#v", tioInfo, i, expArg, gotArg)
			}
		}
	}
	// surplus args
	{
		if !slices.Equal(got.Surplus, exp.Surplus) {
			t.Errorf("tt:%s: surplus args:\nexpected %+#v\n     got %+#v",
				tioInfo, exp.Surplus, got.Surplus)
		}
	}
	// subcommand
	{
		switch {
		case got.Subcmd == nil && exp.Subcmd != nil:
			t.Errorf("tt:%s:\nexpected subcommand %+v\ngot nil", tioInfo, exp.Subcmd)
		case got.Subcmd != nil && exp.Subcmd == nil:
			t.Errorf("tt:%s:\ndid not expect a subcommand\ngot %+v", tioInfo, got.Subcmd)
		case got.Subcmd != nil && exp.Subcmd != nil:
			cmpParsed(t, tioInfo, *exp.Subcmd, *got.Subcmd)
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
		},
		{
			err:      MissingOptionsError{Names: []string{"-a", "--bb"}},
			target:   MissingOptionsError{Names: []string{"-c"}},
			expected: false,
		},
		{
			err:      MissingArgsError{Names: []string{"a", "b"}},
			target:   MissingArgsError{Names: []string{"a", "b"}},
			expected: true,
		},
		{
			err:      MissingArgsError{Names: []string{"a", "b"}},
			target:   MissingArgsError{Names: []string{"c"}},
			expected: false,
		},
		{
			err:      UnknownSubcmdError{Name: "a"},
			target:   UnknownSubcmdError{Name: "a"},
			expected: true,
		},
		{
			err:      UnknownSubcmdError{Name: "c"},
			target:   UnknownSubcmdError{Name: "d"},
			expected: false,
		},
		{
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

func TestLookups(t *testing.T) {
	type testLookup struct {
		id        string
		expExists bool
		expValue  any
	}
	type testInputOutput struct {
		args       []string
		optLookups []testLookup
		argLookups []testLookup
	}
	type testCase struct {
		name       string
		cmd        *RootCommandInfo
		variations []testInputOutput
	}

	for ttIdx, tt := range []testCase{
		0: {
			name: "options_only",
			cmd: NewCmd("optsonly").
				Opt(NewBoolOpt("aa")).
				Opt(NewOpt("bb").ShortOnly('b')).
				Opt(NewOpt("cc").Required()).
				Opt(NewOpt("dd").Default("v4")).
				Opt(NewOpt("ee")).
				Build(),
			variations: []testInputOutput{
				{
					args: []string{"-b", "v2", "--aa", "--cc=v3"},
					optLookups: []testLookup{
						{id: "bb", expExists: true, expValue: "v2"},
						{id: "cc", expExists: true, expValue: "v3"},
						{id: "dd", expExists: true, expValue: "v4"},
						{id: "ee", expExists: false, expValue: nil},
					},
				},
			},
		},
	} {
		for tioIdx, tio := range tt.variations {
			t.Run(fmt.Sprintf("%s %d", tt.name, tioIdx), func(t *testing.T) {
				gotParsed, gotErr := tt.cmd.Parse(tio.args...)
				if gotErr != nil {
					t.Fatalf("tt[%d], tio[%d]: error: %#+v", ttIdx, tioIdx, gotErr)
					return
				}

				for luIdx, lu := range tio.optLookups {
					gotValue, gotExists := gotParsed.LookupOpt(lu.id)
					if gotExists != lu.expExists {
						t.Fatalf("tt[%d], tio[%d], lu[%d]: expected exists %v, got %v", ttIdx, tioIdx, luIdx, lu.expExists, gotExists)
					}

					if !reflect.DeepEqual(gotValue, lu.expValue) {
						t.Fatalf("tt[%d], tio[%d], lu[%d]: expected %v, got %v", ttIdx, tioIdx, luIdx, lu.expValue, gotValue)
					}
					if gotExists {
						if !reflect.DeepEqual(gotParsed.Opt(lu.id), lu.expValue) {
							t.Fatalf("tt[%d], tio[%d], lu[%d]: expected %v, got %v", ttIdx, tioIdx, luIdx, lu.expValue, gotValue)
						}
					}
				}

				for luIdx, lu := range tio.argLookups {
					gotValue, gotExists := gotParsed.LookupArg(lu.id)
					if gotExists != lu.expExists {
						t.Fatalf("tt[%d], tio[%d], lu[%d]: expected exists %v, got %v", ttIdx, tioIdx, luIdx, lu.expExists, gotExists)
					}

					if !reflect.DeepEqual(gotValue, lu.expValue) {
						t.Fatalf("tt[%d], tio[%d], lu[%d]: expected %v, got %v", ttIdx, tioIdx, luIdx, lu.expValue, gotValue)
					}
					if gotExists {
						if !reflect.DeepEqual(gotParsed.Arg(lu.id), lu.expValue) {
							t.Fatalf("tt[%d], tio[%d], lu[%d]: expected %v, got %v", ttIdx, tioIdx, luIdx, lu.expValue, gotValue)
						}
					}
				}
			})
		}
	}
}

func ttCase() string {
	_, _, line, _ := runtime.Caller(1)
	return fmt.Sprintf("%d", line)
}
