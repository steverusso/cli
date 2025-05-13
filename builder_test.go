package cli

import (
	"reflect"
	"testing"
)

func TestBuilder(t *testing.T) {
	for ttIdx, tt := range []struct {
		name         string
		builds       []func()
		expPanicVals []any
	}{
		{
			name: "control test clean ones",
			builds: []func(){
				func() { NewCmd("a") },
				func() { c := NewCmd("a"); c.Parse() },
			},
			expPanicVals: []any{nil, nil},
		},
		{
			name: "mixing positional args and subcommands",
			builds: []func(){
				func() {
					NewCmd("root").
						Subcmd(NewCmd("sc")).
						Arg(NewArg("a1"))
				},
				func() {
					NewCmd("root").
						Arg(NewArg("a1")).
						Subcmd(NewCmd("sc"))
				},
			},
			expPanicVals: []any{
				errMixingPosArgsAndSubcmds,
				errMixingPosArgsAndSubcmds,
			},
		},
		{
			name: "empty command name",
			builds: []func(){
				func() { NewCmd("") },
				func() { NewCmd("root").Subcmd(NewCmd("")) },
			},
			expPanicVals: []any{
				errEmptyCmdName,
				errEmptyCmdName,
			},
		},
		{
			name: "command name with whitespace",
			builds: []func(){
				func() { NewCmd(" ") },
				func() { NewCmd("a b") },
				func() { NewCmd("ab").Subcmd(NewCmd("c\td")) },
			},
			expPanicVals: []any{
				"invalid command name ' ': cannot contain whitespace",
				"invalid command name 'a b': cannot contain whitespace",
				"invalid command name 'c\td': cannot contain whitespace",
			},
		},
		{
			name: "empty input ids",
			builds: []func(){
				func() { NewCmd("root").Opt(NewOpt("")) },
				func() { NewCmd("root").Arg(NewArg("")) },
			},
			expPanicVals: []any{
				errEmptyInputID,
				errEmptyInputID,
			},
		},
		{
			name:         "empty option names",
			builds:       []func(){func() { NewCmd("root").Opt(NewArg("a1")) }},
			expPanicVals: []any{errEmptyOptNames},
		},
		{
			name: "duplicate option ids",
			builds: []func(){
				func() {
					NewCmd("root").
						Opt(NewOpt("o1")).
						Opt(NewOpt("o2")).
						Opt(NewOpt("o3")).
						Subcmd(NewCmd("one").
							Opt(NewOpt("o1")).
							Opt(NewOpt("o2")).
							Opt(NewOpt("o3")).
							Opt(NewOpt("o1")).
							Opt(NewOpt("o5"))).
						ParseOrExit()
				},
				func() {
					NewCmd("root").
						Arg(NewArg("a1")).
						Arg(NewArg("a1")).
						Arg(NewArg("a2")).
						Arg(NewArg("a3")).
						ParseOrExit()
				},
			},
			expPanicVals: []any{
				"command 'root one' contains duplicate option ids 'o1'",
				"command 'root' contains duplicate argument ids 'a1'",
			},
		},
		{
			name: "duplicate option short names",
			builds: []func(){
				func() {
					NewCmd("root").
						Opt(NewOpt("aa").Short('a')).
						Opt(NewOpt("bb").Short('b')).
						ParseOrExit([]string{}...)
				},
				func() {
					NewCmd("root").
						Opt(NewOpt("aa").Short('a')).
						Opt(NewOpt("bb").Short('a')).
						ParseOrExit()
				},
				func() {
					NewCmd("root").
						Opt(NewOpt("aa").ShortOnly('b')).
						Opt(NewOpt("bb").Short('b')).
						ParseOrExit()
				},
			},
			expPanicVals: []any{
				nil,
				"command 'root' contains duplicate option short name 'a'",
				"command 'root' contains duplicate option short name 'b'",
			},
		},
		{
			name: "duplicate option long names",
			builds: []func(){
				func() {
					NewCmd("root").
						Opt(NewOpt("aa").Long("aaa")).
						Opt(NewOpt("bb").Long("bbb")).
						Opt(NewOpt("cc").Long("aaa")).
						ParseOrExit()
				},
			},
			expPanicVals: []any{
				"command 'root' contains duplicate option long name 'aaa'",
			},
		},
		{
			name: "options as positional arguments",
			builds: []func(){
				func() { NewCmd("root").Arg(NewOpt("o1")) },
				func() { NewCmd("root").Subcmd(NewCmd("sc").Arg(NewOpt("o1"))) },
			},
			expPanicVals: []any{
				errOptAsPosArg,
				errOptAsPosArg,
			},
		},
		{
			name: "required positional arguments coming after optional ones",
			builds: []func(){
				func() {
					NewCmd("root").
						Arg(NewArg("a")).
						Arg(NewArg("b").Required())
				},
				func() {
					NewCmd("root").
						Subcmd(NewCmd("subcmd").
							Arg(NewArg("a")).
							Arg(NewArg("b").Required()))
				},
			},
			expPanicVals: []any{
				errReqArgAfterOptional,
				errReqArgAfterOptional,
			},
		},
		{
			name: "duplicate subcommand names",
			builds: []func(){
				func() {
					NewCmd("root").
						Subcmd(NewCmd("bb")).
						Subcmd(NewCmd("bb")).
						ParseOrExit()
				},
				func() {
					NewCmd("root").
						Subcmd(NewCmd("subcmd").
							Subcmd(NewCmd("aa")).
							Subcmd(NewCmd("bb")).
							Subcmd(NewCmd("aa")).
							Subcmd(NewCmd("cc"))).
						ParseOrExit()
				},
			},
			expPanicVals: []any{
				"command 'root' contains duplicate subcommand name 'bb'",
				"command 'root subcmd' contains duplicate subcommand name 'aa'",
			},
		},
	} {
		t.Run("prevent "+tt.name, func(t *testing.T) {
			for i, build := range tt.builds {
				expPanicVal := tt.expPanicVals[i]
				func() {
					defer func() {
						gotPanicVal := recover()
						if !reflect.DeepEqual(gotPanicVal, expPanicVal) {
							t.Fatalf("tt[%d]: panic values don't match\nexpected: %+#v\n     got: %+#v",
								ttIdx, expPanicVal, gotPanicVal)
						}
					}()
					build()
				}()
			}
		})
	}
}
