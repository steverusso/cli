# cli

[![Go Reference](https://pkg.go.dev/badge/github.com/steverusso/cli.svg)](https://pkg.go.dev/github.com/steverusso/cli)

```shell
go get github.com/steverusso/cli@latest
```

## Features

* Lightweight, simple, composable.
* There are **zero** external dependencies.
* No required project / file layout or recommended use of a generator.
* No reflection.
* Inputs can additionally be parsed from environment variables and / or default values.
* Nested subcommands.
* Clean, well-formatted help messages by default.
* Ability to build custom help messages.

> [!NOTE]
> This is primarily a library to parse command line arguments. Anything it offers in
> addition to that (such as optionally reading values from environment variables as well)
> should be invisible to the user who wants to simply parse command line arguments.
