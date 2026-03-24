package main

import (
	"bytes"
	"strings"
	"testing"

	"authproxy/internal/config"
)

func TestRunPrintsHelpAndReturnsNil(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"--help"}, {"-h"}} {
		var stdout bytes.Buffer
		err := run(args, func(string) (string, bool) {
			return "", false
		}, &stdout)
		if err != nil {
			t.Fatalf("run(%v) returned error: %v", args, err)
		}
		if !strings.Contains(stdout.String(), "--upstream-url") {
			t.Fatalf("stdout = %q, want --upstream-url", stdout.String())
		}
		if !strings.Contains(stdout.String(), "-h, --help") {
			t.Fatalf("stdout = %q, want -h, --help", stdout.String())
		}
	}
}

func TestRunReturnsConfigError(t *testing.T) {
	t.Parallel()

	err := run(nil, func(string) (string, bool) {
		return "", false
	}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("run returned nil error, want config error")
	}
	if err.Error() != "missing upstream URL" {
		t.Fatalf("error = %q, want missing upstream URL", err)
	}
}

var _ config.LookupEnv = func(string) (string, bool) { return "", false }
