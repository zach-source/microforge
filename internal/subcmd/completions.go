package subcmd

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed completions_assets/mforge.bash
var bashCompletion string

//go:embed completions_assets/mforge.zsh
var zshCompletion string

//go:embed completions_assets/install.sh
var installScript string

func Completions(_ string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge completions <install|path|bash|zsh>")
	}
	op := args[0]
	switch op {
	case "install":
		shell := detectShell()
		fmt.Printf("source <(mforge completions %s)\n", shell)
		return nil
	case "path":
		shell := detectShell()
		path, err := writeCompletionTemp(shell)
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	case "bash":
		return printCompletion(bashCompletion)
	case "zsh":
		return printCompletion(zshCompletion)
	default:
		return fmt.Errorf("unknown completions subcommand: %s", op)
	}
}

func detectShell() string {
	if strings.TrimSpace(os.Getenv("ZSH_VERSION")) != "" {
		return "zsh"
	}
	shell := strings.ToLower(strings.TrimSpace(os.Getenv("SHELL")))
	if strings.HasSuffix(shell, "/zsh") {
		return "zsh"
	}
	return "bash"
}

func printCompletion(content string) error {
	_, err := io.WriteString(os.Stdout, content)
	return err
}

func writeCompletionTemp(shell string) (string, error) {
	content := bashCompletion
	if shell == "zsh" {
		content = zshCompletion
	}
	name := fmt.Sprintf("mforge-completions-%s-%d", shell, time.Now().UnixNano())
	path := filepath.Join(os.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

var _ = installScript
