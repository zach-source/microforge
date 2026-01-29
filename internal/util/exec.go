// Package util provides common utility functions for file operations and command execution.
package util

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CmdResult holds the captured stdout and stderr from a command execution.
type CmdResult struct{ Stdout, Stderr string }

// Run executes a command and captures its output. If ctx is nil, a 30-second timeout is used.
// Returns an error containing the command, args, and stderr on failure.
func Run(ctx context.Context, name string, args ...string) (CmdResult, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	res := CmdResult{Stdout: outb.String(), Stderr: errb.String()}
	if err != nil {
		trim := strings.TrimSpace(res.Stderr)
		if trim == "" {
			trim = err.Error()
		}
		return res, fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), trim)
	}
	return res, nil
}

// RunInDir executes a command in the specified directory and captures its output.
// If ctx is nil, a 30-second timeout is used.
func RunInDir(ctx context.Context, dir string, name string, args ...string) (CmdResult, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	res := CmdResult{Stdout: outb.String(), Stderr: errb.String()}
	if err != nil {
		trim := strings.TrimSpace(res.Stderr)
		if trim == "" {
			trim = err.Error()
		}
		return res, fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), trim)
	}
	return res, nil
}
