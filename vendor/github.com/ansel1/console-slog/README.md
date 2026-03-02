# console-slog

[![Go Reference](https://pkg.go.dev/badge/github.com/ansel1/console-slog.svg)](https://pkg.go.dev/github.com/ansel1/console-slog) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/ansel1/console-slog/master/LICENSE) [![Build](https://github.com/ansel1/console-slog/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/ansel1/slog-console/actions/workflows/go.yml) [![codecov](https://codecov.io/gh/ansel1/console-slog/graph/badge.svg?token=ZIJT9L79QP)](https://codecov.io/gh/ansel1/console-slog) [![Go Report Card](https://goreportcard.com/badge/github.com/ansel1/console-slog)](https://goreportcard.com/report/github.com/ansel1/console-slog)

A handler for slog that prints colorized logs, similar to zerolog's console writer output without sacrificing performances.

## Installation
```bash
go get github.com/ansel1/console-slog@latest
```

## Example
```go
package main

import (
	"errors"
	"log/slog"
	"os"

	"github.com/ansel1/console-slog"
)

func main() {
	logger := slog.New(
		console.NewHandler(os.Stderr, &console.HandlerOptions{Level: slog.LevelDebug}),
	)
	slog.SetDefault(logger)
	slog.Info("Hello world!", "foo", "bar")
	slog.Debug("Debug message")
	slog.Warn("Warning message")
	slog.Error("Error message", "err", errors.New("the error"))

	logger = logger.With("foo", "bar").
		WithGroup("the-group").
		With("bar", "baz")

	logger.Info("group info", "attr", "value")
}
```

![output](./doc/img/output.png)

When setting `console.HandlerOptions.AddSource` to `true`:
```go
console.NewHandler(os.Stderr, &console.HandlerOptions{Level: slog.LevelDebug, AddSource: true})
```
![output-with-source](./doc/img/output-with-source.png)

## Performances
See [benchmark file](./bench_test.go) for details.

The handler itself performs quite well compared to std-lib's handlers. It does no allocation.  It is generally faster
then slog.TextHandler, and a little slower than slog.JSONHandler.

## Credit

This is a forked and heavily modified variant of github.com/phsym/console-slog.