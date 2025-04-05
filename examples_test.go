package cli_test

import (
	"fmt"
	"image"
	"net/url"
	"strconv"
	"strings"

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
	//     an option
	//
	//   -h, --help
	//     Show this help message and exit.
}

func ExampleDefaultFullHelp() {
	c := cli.NewCmd("eg").
		Help("an example command").
		Opt(cli.NewOpt("aa").Env("AA").Help("an option")).
		Opt(cli.NewOpt("bb").
			Help("another option that is required and has a really long blurb to show how it will be wrapped").
			Required())

	h := cli.DefaultFullHelp(&c)
	fmt.Println(h)
	// Output:
	// eg - an example command
	//
	// usage:
	//   eg [options]
	//
	// options:
	//   --aa  <arg>
	//     an option
	//
	//     [env: AA]
	//
	//   --bb  <arg>   (required)
	//     another option that is required and has a really long blurb to show how it will be
	//     wrapped
	//
	//   -h, --help
	//     Show this help message and exit.
}
