package library

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Doc struct {
	Service string
	Path    string
	Content string
}

type Result struct {
	Service string `json:"service"`
	Path    string `json:"path"`
	Snippet string `json:"snippet"`
}

type QueryRequest struct {
	Service string `json:"service"`
	Query   string `json:"query"`
}

type QueryResponse struct {
	Results []Result `json:"results"`
	Source  string   `json:"source"`
}

type Config struct {
	Docs          []string
	Context7URL   string
	Context7Token string
}

type Library struct {
	Docs   []Doc
	Config Config
}

func New(cfg Config) (*Library, error) {
	docs, err := loadDocs(cfg.Docs)
	if err != nil {
		return nil, err
	}
	return &Library{Docs: docs, Config: cfg}, nil
}

func loadDocs(paths []string) ([]Doc, error) {
	var out []Doc
	for _, root := range paths {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if _, err := os.Stat(root); err != nil {
			continue
		}
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".md" && ext != ".txt" {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			service := serviceFromPath(root, path)
			out = append(out, Doc{Service: service, Path: path, Content: string(b)})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func serviceFromPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Base(root)
	}
	parts := strings.Split(rel, string(os.PathSeparator))
	if len(parts) == 0 {
		return filepath.Base(root)
	}
	if parts[0] == "services" && len(parts) > 1 {
		return parts[1]
	}
	return parts[0]
}

func (l *Library) Query(service, query string) (QueryResponse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return QueryResponse{Results: []Result{}, Source: "local"}, nil
	}
	needle := strings.ToLower(query)
	var results []Result
	for _, doc := range l.Docs {
		if service != "" && doc.Service != service {
			continue
		}
		if strings.Contains(strings.ToLower(doc.Content), needle) {
			snippet := makeSnippet(doc.Content, needle)
			results = append(results, Result{Service: doc.Service, Path: doc.Path, Snippet: snippet})
			if len(results) >= 20 {
				break
			}
		}
	}
	if len(results) == 0 && l.Config.Context7URL != "" {
		res, err := queryContext7(l.Config, service, query)
		if err == nil {
			return res, nil
		}
	}
	return QueryResponse{Results: results, Source: "local"}, nil
}

func makeSnippet(content, needle string) string {
	lower := strings.ToLower(content)
	idx := strings.Index(lower, needle)
	if idx < 0 {
		return ""
	}
	start := idx - 80
	if start < 0 {
		start = 0
	}
	end := idx + 120
	if end > len(content) {
		end = len(content)
	}
	return strings.TrimSpace(content[start:end])
}

func queryContext7(cfg Config, service, query string) (QueryResponse, error) {
	payload := map[string]string{"service": service, "query": query}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", cfg.Context7URL, bytes.NewReader(b))
	if err != nil {
		return QueryResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.Context7Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Context7Token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return QueryResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return QueryResponse{}, fmt.Errorf("context7 status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var out QueryResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return QueryResponse{}, err
	}
	out.Source = "context7"
	return out, nil
}
