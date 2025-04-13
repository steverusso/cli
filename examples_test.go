package cli_test

import (
	"fmt"
	"image"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/steverusso/cli"
)

func ExampleInput_Short() {
	c := cli.NewCmd("eg").
		Opt(cli.NewOpt("flag").Short('f')).
		Build()

	p1 := c.ParseOrExit("-f", "hello")
	fmt.Println(p1.Opt("flag"))

	p2 := c.ParseOrExit("--flag", "world")
	fmt.Println(p2.Opt("flag"))
	// Output:
	// hello
	// world
}

func ExampleInput_ShortOnly() {
	c := cli.NewCmd("eg").
		Opt(cli.NewOpt("flag").ShortOnly('f')).
		Build()

	p := c.ParseOrExit("-f", "hello")
	fmt.Println(p.Opt("flag"))

	_, err := c.Parse("--flag", "helloworld")
	fmt.Println(err)
	// Output:
	// hello
	// unknown option '--flag'
}

func ExampleInput_WithParser() {
	pointParser := func(s string) (any, error) {
		comma := strings.IndexByte(s, ',')
		x, _ := strconv.Atoi(s[:comma])
		y, _ := strconv.Atoi(s[comma+1:])
		return image.Point{X: x, Y: y}, nil
	}

	p := cli.NewCmd("cp").
		Opt(cli.NewOpt("aa").WithParser(pointParser)).
		Build().
		ParseOrExit("--aa", "3,7")

	fmt.Printf("%+#v\n", p.Opt("aa"))
	// Output:
	// image.Point{X:3, Y:7}
}

func ExampleParseURL() {
	cmd := cli.NewCmd("url").
		Opt(cli.NewOpt("u").WithParser(cli.ParseURL)).
		Build()

	p := cmd.ParseOrExit("-u", "https://pkg.go.dev/github.com/steverusso/cli#ParseURL")
	fmt.Println(p.Opt("u").(*url.URL))

	_, err := cmd.Parse("-u", "b@d://.com")
	fmt.Println(err)
	// Output:
	// https://pkg.go.dev/github.com/steverusso/cli#ParseURL
	// parsing option 'u': parse "b@d://.com": first path segment in URL cannot contain colon
}

func ExampleParseDuration() {
	cmd := cli.NewCmd("duration").
		Opt(cli.NewOpt("d").WithParser(cli.ParseDuration)).
		Build()

	p := cmd.ParseOrExit("-d", "1h2m3s")
	fmt.Println(p.Opt("d").(time.Duration))

	_, err := cmd.Parse("-d", "not_a_duration")
	fmt.Println(err)
	// Output:
	// 1h2m3s
	// parsing option 'd': time: invalid duration "not_a_duration"
}

func ExampleNewTimeParser() {
	cmd := cli.NewCmd("times").
		Opt(cli.NewOpt("t").WithParser(cli.NewTimeParser("2006-01-02"))).
		Build()

	p := cmd.ParseOrExit("-t", "2025-04-12")
	fmt.Println(p.Opt("t").(time.Time))

	_, err := cmd.Parse("-t", "hello")
	fmt.Println(err)
	// Output:
	// 2025-04-12 00:00:00 +0000 UTC
	// parsing option 't': parsing time "hello" as "2006-01-02": cannot parse "hello" as "2006"
}

func ExampleCommand_HelpUsage() {
	_, err := cli.NewCmd("eg").
		Help("an example command").
		HelpUsage(
			"eg [--aa <arg>]",
			"eg [-h]",
		).
		Opt(cli.NewOpt("aa").Help("an option")).
		Build().
		Parse("--help")

	fmt.Println(err)
	// Output:
	// eg - an example command
	//
	// usage:
	//   eg [--aa <arg>]
	//   eg [-h]
	//
	// options:
	//   --aa  <arg>
	//       an option
	//
	//   -h, --help
	//       Show this help message and exit.
}

func ExampleDefaultFullHelp() {
	c := cli.NewCmd("eg").
		Help("an example command").
		Opt(cli.NewOpt("aa").Env("AA").Default("def").Help("an option")).
		Opt(cli.NewOpt("bb").
			Help("another option that is required and has a really long blurb to show how it will be wrapped").
			Required()).
		Arg(cli.NewArg("cc").Help("a positional argument"))

	h := cli.DefaultFullHelp(&c)
	fmt.Println(h)
	// Output:
	// eg - an example command
	//
	// usage:
	//   eg [options] [arguments]
	//
	// options:
	//   --aa  <arg>
	//       an option
	//
	//       [default: def]
	//       [env: AA]
	//
	//   --bb  <arg>   (required)
	//       another option that is required and has a really long blurb to show how it will be
	//       wrapped
	//
	//   -h, --help
	//       Show this help message and exit.
	//
	// arguments:
	//   [cc]
	//       a positional argument
}

func ExampleDefaultShortHelp_simple() {
	c := cli.NewCmd("eg").
		Help("an example command").
		Opt(cli.NewOpt("aa").Short('a').Env("AA").Default("def").Help("an option")).
		Opt(cli.NewOpt("bb").Short('b').Required().Help("another option")).
		Arg(cli.NewArg("cc").Required().Help("a required positional argument")).
		Arg(cli.NewArg("dd").Env("PA2").Help("an optional positional argument"))

	h := cli.DefaultShortHelp(&c)
	fmt.Println(h)
	// Output:
	// eg - an example command
	//
	// usage:
	//   eg [options] [arguments]
	//
	// options:
	//   -a, --aa  <arg>   an option (default: def) [$AA]
	//   -b, --bb  <arg>   another option (required)
	//   -h, --help        Show this help message and exit.
	//
	// arguments:
	//   <cc>   a required positional argument (required)
	//   [dd]   an optional positional argument [$PA2]
}

func ExampleDefaultShortHelp_simpleWithSubcommands() {
	c := cli.NewCmd("eg").
		Help("an example command").
		Opt(cli.NewOpt("aa").Short('a').Env("AA").Default("def").Help("an option")).
		Opt(cli.NewOpt("bb").Short('b').Required().Help("another option")).
		Subcmd(cli.NewCmd("subcommand1").Help("a subcommand")).
		Subcmd(cli.NewCmd("subcommand2").Help("another subcommand"))

	h := cli.DefaultShortHelp(&c)
	fmt.Println(h)
	// Output:
	// eg - an example command
	//
	// usage:
	//   eg [options] <command>
	//
	// options:
	//   -a, --aa  <arg>   an option (default: def) [$AA]
	//   -b, --bb  <arg>   another option (required)
	//   -h, --help        Show this help message and exit.
	//
	// commands:
	//    subcommand1   a subcommand
	//    subcommand2   another subcommand
}

func ExampleDefaultShortHelp_complex() {
	c := cli.NewCmd("eg").
		Help("an example command").
		Opt(cli.NewOpt("aa").Env("AA").Default("def").Help("an option")).
		Opt(cli.NewOpt("bb").
			Help("another option that is required and has a really long blurb to show how it will be wrapped").
			Required()).
		Opt(cli.NewOpt("short-blurb-but-really-long-name").Help("another option")).
		Arg(cli.NewArg("posarg1").Required().Help("a required positional argument")).
		Arg(cli.NewArg("posarg2").Env("PA2").Help("an optional positional argument"))

	h := cli.DefaultShortHelp(&c)
	fmt.Println(h)
	// Output:
	// eg - an example command
	//
	// usage:
	//   eg [options] [arguments]
	//
	// options:
	//       --aa  <arg>   an option (default: def) [$AA]
	//       --bb  <arg>   another option that is required and has a really long blurb to show
	//                     how it will be wrapped (required)
	//   -h, --help        Show this help message and exit.
	//       --short-blurb-but-really-long-name  <arg>
	//                     another option
	//
	// arguments:
	//   <posarg1>   a required positional argument (required)
	//   [posarg2]   an optional positional argument [$PA2]
}
