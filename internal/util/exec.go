package util

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type CmdResult struct{ Stdout, Stderr string }

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
