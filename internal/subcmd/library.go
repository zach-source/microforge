package subcmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/example/microforge/internal/library"
	"github.com/example/microforge/internal/store"
)

func Library(home string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mforge library <start|query> ...")
	}
	op := args[0]
	rest := args[1:]

	switch op {
	case "start":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge library start <rig> [--addr <addr>]")
		}
		rigName := rest[0]
		var addr string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--addr":
				if i+1 < len(rest) {
					addr = rest[i+1]
					i++
				}
			}
		}
		cfg, err := store.LoadRigConfig(store.RigConfigPath(home, rigName))
		if err != nil {
			return err
		}
		if addr == "" {
			addr = cfg.LibraryAddr
		}
		lib, err := library.New(library.Config{Docs: cfg.LibraryDocs, Context7URL: cfg.LibraryContext7URL, Context7Token: cfg.LibraryContext7Token})
		if err != nil {
			return err
		}
		h := http.NewServeMux()
		h.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
			var req library.QueryRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			resp, err := lib.Query(req.Service, req.Query)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			_ = json.NewEncoder(w).Encode(resp)
		})
		fmt.Printf("Library MCP server listening on %s\n", addr)
		return http.ListenAndServe(addr, h)

	case "query":
		if len(rest) < 1 {
			return fmt.Errorf("usage: mforge library query <rig> --q <query> [--service <name>] [--addr <addr>]")
		}
		rigName := rest[0]
		var query, service, addr string
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--q":
				if i+1 < len(rest) {
					query = rest[i+1]
					i++
				}
			case "--service":
				if i+1 < len(rest) {
					service = rest[i+1]
					i++
				}
			case "--addr":
				if i+1 < len(rest) {
					addr = rest[i+1]
					i++
				}
			}
		}
		if strings.TrimSpace(query) == "" {
			return fmt.Errorf("--q is required")
		}
		cfg, err := store.LoadRigConfig(store.RigConfigPath(home, rigName))
		if err != nil {
			return err
		}
		if addr == "" {
			addr = cfg.LibraryAddr
		}
		body, _ := json.Marshal(library.QueryRequest{Service: service, Query: query})
		resp, err := http.Post("http://"+addr+"/query", "application/json", bytes.NewReader(body))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("library query failed: %s", resp.Status)
		}
		var out library.QueryResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return err
		}
		for _, r := range out.Results {
			fmt.Printf("%s\t%s\n", r.Service, r.Path)
			fmt.Println(r.Snippet)
		}
		return nil

	default:
		return fmt.Errorf("unknown library subcommand: %s", op)
	}
}
