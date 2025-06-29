package cli

import "testing"

func TestDefaultHelps(t *testing.T) {
	for _, tt := range []struct {
		Case          string
		cmdInfo       CommandInfo
		expectedShort string
		expectedFull  string
	}{
		{
			Case: ttCase(),
			cmdInfo: New().
				Opt(NewOpt("lorem").
					Short('l').
					Required().
					Help("ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.")).
				Opt(NewOpt("enim-ad-minim").
					Required().
					Help("veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.")),
			expectedShort: `cli.test - 

usage:
  cli.test [options]

options:
  --enim-ad-minim  <arg>
      veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo
      consequat. (required)
  -h, --help
      Show this help message and exit.
  -l, --lorem  <arg>
      ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt
      ut labore et dolore magna aliqua. (required)
`,
			expectedFull: `cli.test - 

usage:
  cli.test [options]

options:
  --enim-ad-minim  <arg>   (required)
      veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo
      consequat.

  -h, --help
      Show this help message and exit.

  -l, --lorem  <arg>   (required)
      ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt
      ut labore et dolore magna aliqua.
`,
		},
	} {
		// Due to the current design, we have to call this in order to get the default
		// help option inserted in (if necessary).
		tt.cmdInfo.prepareAndValidate()

		gotShort := DefaultShortHelp(&tt.cmdInfo)
		if gotShort != tt.expectedShort {
			t.Errorf("%s:short helps don't match\nexpected:\n%s\ngot:\n%s", tt.Case, tt.expectedShort, gotShort)
		}

		gotFull := DefaultFullHelp(&tt.cmdInfo)
		if gotFull != tt.expectedFull {
			t.Errorf("%s: full helps don't match\nexpected:\n%s\ngot:\n%s", tt.Case, tt.expectedFull, gotFull)
		}
	}
}
