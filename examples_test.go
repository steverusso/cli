package cli_test

import (
	"fmt"
	"image"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/steverusso/cli"
)

func ExampleInputInfo_Short() {
	in := cli.New().
		Opt(cli.NewOpt("flag").Short('f'))

	c1 := in.ParseTheseOrExit("-f", "hello")
	fmt.Println(cli.Get[string](c1, "flag"))

	c2 := in.ParseTheseOrExit("--flag", "world")
	fmt.Println(cli.Get[string](c2, "flag"))
	// Output:
	// hello
	// world
}

func ExampleInputInfo_ShortOnly() {
	in := cli.New().
		Opt(cli.NewOpt("flag").ShortOnly('f'))

	c := in.ParseTheseOrExit("-f", "hello")
	fmt.Println(cli.Get[string](c, "flag"))

	_, err := in.ParseThese("--flag", "helloworld")
	fmt.Println(err)
	// Output:
	// hello
	// unknown option '--flag'
}

func ExampleInputInfo_Help() {
	in := cli.New("example").
		Help("example program").
		Opt(cli.NewOpt("aa").Help("a short one or two sentence blurb"))

	_, err := in.ParseThese("-h")
	fmt.Println(err)
	// Output:
	// example - example program
	//
	// usage:
	//   example [options]
	//
	// options:
	//       --aa  <arg>   a short one or two sentence blurb
	//   -h, --help        Show this help message and exit.
}

func ExampleInputInfo_WithValueName_option() {
	in := cli.New("example").
		Help("example program").
		Opt(cli.NewOpt("aa").WithValueName("str").Help("it says '<str>' above instead of '<arg>'"))

	_, err := in.ParseThese("--help")
	fmt.Println(err)
	// Output:
	// example - example program
	//
	// usage:
	//   example [options]
	//
	// options:
	//   --aa  <str>
	//       it says '<str>' above instead of '<arg>'
	//
	//   -h, --help
	//       Show this help message and exit.
}

func ExampleInputInfo_WithValueName_positionalArgument() {
	in := cli.New("example").
		Help("example program").
		Arg(cli.NewArg("aa").WithValueName("filename").Help("it says '[filename]' above instead of '[aa]'"))

	_, err := in.ParseThese("--help")
	fmt.Println(err)
	// Output:
	// example - example program
	//
	// usage:
	//   example [options] [arguments]
	//
	// options:
	//   -h, --help
	//       Show this help message and exit.
	//
	// arguments:
	//   [filename]
	//       it says '[filename]' above instead of '[aa]'
}

func ExampleInputInfo_WithParser() {
	pointParser := func(s string) (any, error) {
		comma := strings.IndexByte(s, ',')
		x, _ := strconv.Atoi(s[:comma])
		y, _ := strconv.Atoi(s[comma+1:])
		return image.Point{X: x, Y: y}, nil
	}

	c := cli.New().
		Opt(cli.NewOpt("aa").WithParser(pointParser)).
		ParseTheseOrExit("--aa", "3,7")

	fmt.Printf("%+#v\n", cli.Get[image.Point](c, "aa"))
	// Output:
	// image.Point{X:3, Y:7}
}

func ExampleInputInfo_Env() {
	os.Setenv("FLAG", "hello")

	c := cli.New().
		Opt(cli.NewOpt("flag").Env("FLAG")).
		ParseTheseOrExit()

	fmt.Println(cli.Get[string](c, "flag"))
	// Output:
	// hello
}

func ExampleInputInfo_Long() {
	in := cli.New("example").
		Help("example program").
		Opt(cli.NewOpt("id1").Help("long name is the id by default")).
		Opt(cli.NewOpt("id2").Long("long-name").Help("long name is set to something other than the id"))

	_, err := in.ParseThese("-h")
	fmt.Println(err)
	// Output:
	// example - example program
	//
	// usage:
	//   example [options]
	//
	// options:
	//   -h, --help               Show this help message and exit.
	//       --id1  <arg>         long name is the id by default
	//       --long-name  <arg>   long name is set to something other than the id
}

func ExampleParseURL() {
	in := cli.New().
		Opt(cli.NewOpt("u").WithParser(cli.ParseURL))

	c := in.ParseTheseOrExit("-u", "https://pkg.go.dev/github.com/steverusso/cli#ParseURL")
	fmt.Println(cli.Get[*url.URL](c, "u"))

	_, err := in.ParseThese("-u", "b@d://.com")
	fmt.Println(err)
	// Output:
	// https://pkg.go.dev/github.com/steverusso/cli#ParseURL
	// parsing option 'u': parse "b@d://.com": first path segment in URL cannot contain colon
}

func ExampleParseDuration() {
	in := cli.New().
		Opt(cli.NewOpt("d").WithParser(cli.ParseDuration))

	c := in.ParseTheseOrExit("-d", "1h2m3s")
	fmt.Println(cli.Get[time.Duration](c, "d"))

	_, err := in.ParseThese("-d", "not_a_duration")
	fmt.Println(err)
	// Output:
	// 1h2m3s
	// parsing option 'd': time: invalid duration "not_a_duration"
}

func ExampleNewTimeParser() {
	in := cli.New().
		Opt(cli.NewOpt("t").WithParser(cli.NewTimeParser("2006-01-02")))

	c := in.ParseTheseOrExit("-t", "2025-04-12")
	fmt.Println(cli.Get[time.Time](c, "t"))

	_, err := in.ParseThese("-t", "hello")
	fmt.Println(err)
	// Output:
	// 2025-04-12 00:00:00 +0000 UTC
	// parsing option 't': parsing time "hello" as "2006-01-02": cannot parse "hello" as "2006"
}

func ExampleNewFileParser() {
	in := cli.New().
		Opt(cli.NewOpt("i").WithParser(cli.NewFileParser(cli.ParseInt))).
		Opt(cli.NewOpt("s").WithParser(cli.NewFileParser(nil)))

	c, _ := in.ParseThese(
		"-i", "testdata/sample_int",
		"-s", "testdata/sample_int",
	)

	fmt.Println(cli.Get[int](c, "i"))
	fmt.Printf("%q\n", cli.Get[string](c, "s"))

	_, err := in.ParseThese("-i", "testdata/sample_empty")
	fmt.Println(err)

	_, err = in.ParseThese("-i", "path_that_doesnt_exist")
	fmt.Println(err)
	// Output:
	// 12345
	// "12345"
	// parsing option 'i': invalid syntax
	// parsing option 'i': open path_that_doesnt_exist: no such file or directory
}

func ExampleCommandInfo_Arg() {
	c := cli.New().
		Arg(cli.NewArg("name")).
		ParseTheseOrExit("alice")

	fmt.Println(cli.Get[string](c, "name"))
	// Output:
	// alice
}

func ExampleCommandInfo_Opt() {
	c := cli.New().
		Opt(cli.NewOpt("a")).
		ParseTheseOrExit("-a", "hello")

	fmt.Println(cli.Get[string](c, "a"))
	// Output:
	// hello
}

func ExampleCommandInfo_ExtraHelp() {
	in := cli.New("example").
		Help("an example command").
		ExtraHelp("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.")

	_, err := in.ParseThese("--help")
	fmt.Println(err)
	// Output:
	// example - an example command
	//
	// overview:
	//   Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor
	//   incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud
	//   exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.
	//
	// usage:
	//   example [options]
	//
	// options:
	//   -h, --help
	//       Show this help message and exit.
}

func ExampleInputInfo_Required_option() {
	in := cli.New().
		Opt(cli.NewOpt("a")).
		Opt(cli.NewOpt("b").Required())

	c, _ := in.ParseThese("-a", "hello", "-b", "world")
	fmt.Println(
		cli.Get[string](c, "a"),
		cli.Get[string](c, "b"),
	)

	_, err := in.ParseThese()
	fmt.Println(err)
	// Output:
	// hello world
	// missing the following required options: -b
}

func ExampleInputInfo_Required_postionalArgument() {
	in := cli.New().
		Arg(cli.NewArg("a").Required()).
		Arg(cli.NewArg("b"))

	c, _ := in.ParseThese("hello", "world")
	fmt.Println(
		cli.Get[string](c, "a"),
		cli.Get[string](c, "b"),
	)

	_, err := in.ParseThese()
	fmt.Println(err)
	// Output:
	// hello world
	// missing the following required arguments: a
}

func ExampleCommandInfo_Usage() {
	in := cli.New("example").
		Help("an example command").
		Usage(
			"example [--aa <arg>]",
			"example [-h]",
		).
		Opt(cli.NewOpt("aa").Help("an option"))

	_, err := in.ParseThese("--help")
	fmt.Println(err)
	// Output:
	// example - an example command
	//
	// usage:
	//   example [--aa <arg>]
	//   example [-h]
	//
	// options:
	//   --aa  <arg>
	//       an option
	//
	//   -h, --help
	//       Show this help message and exit.
}

func ExampleDefaultFullHelp() {
	in := cli.New("example").
		Help("an example command").
		Opt(cli.NewOpt("aa").Env("AA").Default("def").Help("an option")).
		Opt(cli.NewOpt("bb").
			Help("another option that is required and has a really long blurb to show how it will be wrapped").
			Required()).
		Arg(cli.NewArg("cc").Help("a positional argument")).
		Opt(cli.NewBoolOpt("h").
			Help("will show how this command looks in the default full help message").
			WithHelpGen(func(_ cli.Input, c *cli.CommandInfo) string {
				return cli.DefaultFullHelp(c)
			}))

	_, err := in.ParseThese("-h")
	fmt.Println(err)
	// Output:
	// example - an example command
	//
	// usage:
	//   example [options] [arguments]
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
	//   -h
	//       will show how this command looks in the default full help message
	//
	// arguments:
	//   [cc]
	//       a positional argument
}

func ExampleDefaultShortHelp_simple() {
	in := cli.New("example").
		Help("an example command").
		Opt(cli.NewOpt("aa").Short('a').Env("AA").Default("def").Help("an option")).
		Opt(cli.NewOpt("bb").Short('b').Required().Help("another option")).
		Arg(cli.NewArg("cc").Required().Help("a required positional argument")).
		Arg(cli.NewArg("dd").Env("PA2").Help("an optional positional argument")).
		Opt(cli.NewBoolOpt("h").
			Help("will show the default short help message").
			WithHelpGen(func(_ cli.Input, c *cli.CommandInfo) string {
				return cli.DefaultShortHelp(c)
			}))

	_, err := in.ParseThese("-h")
	fmt.Println(err)
	// Output:
	// example - an example command
	//
	// usage:
	//   example [options] [arguments]
	//
	// options:
	//   -a, --aa  <arg>   an option (default: def) [$AA]
	//   -b, --bb  <arg>   another option (required)
	//   -h                will show the default short help message
	//
	// arguments:
	//   <cc>   a required positional argument (required)
	//   [dd]   an optional positional argument [$PA2]
}

func ExampleDefaultShortHelp_simpleWithSubcommands() {
	in := cli.New("example").
		Help("an example command").
		Opt(cli.NewOpt("aa").Short('a').Env("AA").Default("def").Help("an option")).
		Opt(cli.NewOpt("bb").Short('b').Required().Help("another option")).
		Subcmd(cli.NewCmd("subcommand1").Help("a subcommand")).
		Subcmd(cli.NewCmd("subcommand2").Help("another subcommand")).
		Opt(cli.NewBoolOpt("h").
			Help("will show the default short help message").
			WithHelpGen(func(_ cli.Input, c *cli.CommandInfo) string {
				return cli.DefaultShortHelp(c)
			}))

	_, err := in.ParseThese("-h")
	fmt.Println(err)
	// Output:
	// example - an example command
	//
	// usage:
	//   example [options] <command>
	//
	// options:
	//   -a, --aa  <arg>   an option (default: def) [$AA]
	//   -b, --bb  <arg>   another option (required)
	//   -h                will show the default short help message
	//
	// commands:
	//    subcommand1   a subcommand
	//    subcommand2   another subcommand
}

func ExampleDefaultShortHelp_complex() {
	in := cli.New("example").
		Help("an example command").
		Opt(cli.NewOpt("aa").Env("AA").Default("def").Help("an option")).
		Opt(cli.NewOpt("bb").
			Help("another option that is required and has a really long blurb to show how it will be wrapped").
			Required()).
		Opt(cli.NewOpt("kind-of-a-long-name").
			Help("due to this option's name, the blurbs for each option on this command " +
				"will begin on their own non-indented lines")).
		Arg(cli.NewArg("posarg1").Required().Help("a required positional argument")).
		Arg(cli.NewArg("posarg2").Env("PA2").Help("an optional positional argument")).
		Opt(cli.NewBoolOpt("h").
			Help("will show the default short help message").
			WithHelpGen(func(_ cli.Input, c *cli.CommandInfo) string {
				return cli.DefaultShortHelp(c)
			}))

	_, err := in.ParseThese("-h")
	fmt.Println(err)
	// Output:
	// example - an example command
	//
	// usage:
	//   example [options] [arguments]
	//
	// options:
	//   --aa  <arg>
	//       an option (default: def) [$AA]
	//   --bb  <arg>
	//       another option that is required and has a really long blurb to show how it will be
	//       wrapped (required)
	//   -h
	//       will show the default short help message
	//   --kind-of-a-long-name  <arg>
	//       due to this option's name, the blurbs for each option on this command will begin on
	//       their own non-indented lines
	//
	// arguments:
	//   <posarg1>   a required positional argument (required)
	//   [posarg2]   an optional positional argument [$PA2]
}

func ExampleGet_option() {
	c := cli.New().
		Opt(cli.NewOpt("a")).
		Opt(cli.NewOpt("b")).
		ParseTheseOrExit("-b=hello")

	// The following line would panic because 'a' isn't present.
	// a := cli.Get[string](c, "a")

	b := cli.Get[string](c, "b")
	fmt.Printf("b: %q\n", b)
	// Output:
	// b: "hello"
}

func ExampleGetOr_option() {
	c := cli.New().
		Opt(cli.NewOpt("a")).
		Opt(cli.NewOpt("b")).
		ParseTheseOrExit("-a=hello")

	a := cli.Get[string](c, "a")
	b := cli.GetOr(c, "b", "world")

	fmt.Printf("a: %q\n", a)
	fmt.Printf("b: %q\n", b)
	// Output:
	// a: "hello"
	// b: "world"
}

func ExampleLookup_option() {
	c := cli.New().
		Opt(cli.NewOpt("a")).
		Opt(cli.NewOpt("b")).
		ParseTheseOrExit("-b=hello")

	a, hasA := cli.Lookup[string](c, "a")
	b, hasB := cli.Lookup[string](c, "b")

	fmt.Printf("a: %q, %v\n", a, hasA)
	fmt.Printf("b: %q, %v\n", b, hasB)
	// Output:
	// a: "", false
	// b: "hello", true
}

func ExampleGet_positionalArgs() {
	c := cli.New().
		Arg(cli.NewArg("a")).
		Arg(cli.NewArg("b")).
		ParseTheseOrExit("hello")

	// The following line would panic because 'a' isn't present.
	// b := cli.Get[string](c, "b")

	a := cli.Get[string](c, "a")
	fmt.Printf("a: %q\n", a)
	// Output:
	// a: "hello"
}

func ExampleGetOr_positionlArgs() {
	c := cli.New().
		Arg(cli.NewArg("a")).
		Arg(cli.NewArg("b")).
		ParseTheseOrExit("hello")

	a := cli.Get[string](c, "a")
	b := cli.GetOr(c, "b", "world")

	fmt.Printf("a: %q\n", a)
	fmt.Printf("b: %q\n", b)
	// Output:
	// a: "hello"
	// b: "world"
}

func ExampleGetOrFunc() {
	c := cli.New().
		Opt(cli.NewOpt("a")).
		Opt(cli.NewOpt("b")).
		ParseTheseOrExit("-a", "hello")

	a := cli.GetOr(c, "a", "")
	b := cli.GetOrFunc(c, "b", func() string {
		return "world"
	})

	fmt.Printf("a: %q\n", a)
	fmt.Printf("b: %q\n", b)
	// Output:
	// a: "hello"
	// b: "world"
}

func ExampleLookup_positionalArgs() {
	c := cli.New().
		Arg(cli.NewArg("a")).
		Arg(cli.NewArg("b")).
		ParseTheseOrExit("hello")

	a, hasA := cli.Lookup[string](c, "a")
	b, hasB := cli.Lookup[string](c, "b")

	fmt.Printf("a: %q, %v\n", a, hasA)
	fmt.Printf("b: %q, %v\n", b, hasB)
	// Output:
	// a: "hello", true
	// b: "", false
}

func ExampleGetAll() {
	c := cli.New().
		Opt(cli.NewIntOpt("a").Default("0")).
		ParseTheseOrExit("-a", "1", "-a", "2", "-a", "3")

	a := cli.GetAll[int](c, "a")

	fmt.Printf("%+#v\n", a)
	// Output:
	// []int{0, 1, 2, 3}
}

func ExampleGetAllSeq() {
	c := cli.New().
		Opt(cli.NewOpt("a").Default("Lorem")).
		ParseTheseOrExit(
			"-a", "ipsum",
			"-a", "dolor",
			"-a", "sit",
			"-a", "amet")

	for str := range cli.GetAllSeq[string](c, "a") {
		fmt.Printf("%v\n", str)
	}
	// Output:
	// Lorem
	// ipsum
	// dolor
	// sit
	// amet
}
