package ualog_test

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/gopcua/opcua/ualog"
)

func ExampleNew() {
	handler := slog.NewTextHandler(os.Stdout, testLogOpts())
	ctx := ualog.New(context.Background(), ualog.WithHandler(handler))

	ualog.Info(ctx, "hello, world!")

	// Output: level=INFO msg="hello, world!"
}

func ExampleNew_json() {
	handler := slog.NewJSONHandler(os.Stdout, testLogOpts())
	ctx := ualog.New(context.Background(), ualog.WithHandler(handler))

	ualog.Info(ctx, "hello, world!")

	// Output: {"level":"INFO","msg":"hello, world!"}
}

func ExampleNew_fromlogger() {
	handler := slog.NewJSONHandler(os.Stdout, testLogOpts())
	logger := slog.New(handler).With("foo", "bar")

	ctx := ualog.New(context.Background(), ualog.WithLogger(logger))

	ualog.Info(ctx, "hello, world!")

	// Output: {"level":"INFO","msg":"hello, world!","foo":"bar"}
}

func ExampleError() {
	handler := slog.NewJSONHandler(os.Stdout, testLogOpts())
	ctx := ualog.New(context.Background(), ualog.WithHandler(handler))

	err := errors.New("critical reactor core failure")
	ualog.Error(ctx, "whoopsi daisies", ualog.Err(err))

	// {"level":"ERROR","msg":"whoopsi daisies","err":"critical reactor core failure"}
}

func ExampleWithErrorKey() {
	handler := slog.NewTextHandler(os.Stdout, testLogOpts())
	ctx := ualog.New(context.Background(), ualog.WithHandler(handler),
		ualog.WithErrorKey("oops"),
	)

	err := errors.New("unknown error")
	ualog.Error(ctx, "something went wrong", ualog.Err(err))

	// Output: level=ERROR msg="something went wrong" oops="unknown error"
}

// testLogOpts returns a handler options instance that removes the time
// from log records to make the test output predictable
func testLogOpts() *slog.HandlerOptions {
	return &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			return a
		},
	}
}
