package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type object = map[string]any

type issue struct {
	Number int    `json:"number"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
}

type repository struct {
	mu          sync.RWMutex
	ID          int
	Owner       string
	Name        string
	Description string
	Private     bool
	Issues      []issue
	Labels      map[string]bool
	Refs        map[string]string
	Files       map[string][]byte
	PRs         []object
}

type forge struct {
	mu     sync.RWMutex
	repos  map[string]*repository
	serial int
}

func newForge() *forge { return &forge{repos: make(map[string]*repository)} }

func (f *forge) find(owner, name string) *repository {
	f.mu.RLock()
	r := f.repos[owner+"/"+name]
	f.mu.RUnlock()
	return r
}

func (f *forge) create(owner, name string) (*repository, bool) {
	if name == "" || strings.ContainsAny(name, "/\\ ") {
		return nil, false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	key := owner + "/" + name
	if f.repos[key] != nil {
		return nil, false
	}
	f.serial++
	r := &repository{ID: f.serial, Owner: owner, Name: name, Labels: make(map[string]bool), Refs: make(map[string]string), Files: make(map[string][]byte)}
	f.repos[key] = r
	return r, true
}

func (r *repository) json() object {
	return object{"id": r.ID, "node_id": fmt.Sprintf("R_%d", r.ID), "name": r.Name, "full_name": r.Owner + "/" + r.Name,
		"owner": object{"login": r.Owner}, "description": r.Description, "private": r.Private, "default_branch": "main",
		"has_issues": true, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z"}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-GitHub-Request-Id", "R5-FORGE")
	w.WriteHeader(status)
	if status != http.StatusNoContent {
		_ = json.NewEncoder(w).Encode(value)
	}
}

func readObject(r *http.Request) (object, error) {
	defer r.Body.Close()
	data, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	v := object{}
	if len(data) == 0 {
		return v, nil
	}
	if err = json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func text(v any) string { s, _ := v.(string); return s }

func (f *forge) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("X-GitHub-Request-Id", "R5-FORGE")
	if req.URL.Path == "/healthz" {
		writeJSON(w, 200, object{"ok": true})
		return
	}
	if req.URL.Path == "/" {
		http.Redirect(w, req, "/", 200)
		return
	}
	if strings.HasPrefix(req.URL.Path, "/api/") || req.URL.Path == "/mcp" {
		secret := os.Getenv("MAKEFUN_SHOP_TOKEN")
		if secret == "" {
			secret = "forge-demo"
		}
		if req.Header.Get("Authorization") != "Bearer "+secret && req.Header.Get("Authorization") != "token "+secret {
			writeJSON(w, 401, object{"message": "Bad credentials"})
			return
		}
		in, err := readObject(req)
		if err != nil {
			writeJSON(w, 400, object{"message": "Malformed JSON"})
			return
		}
		if req.URL.Path == "/mcp" {
			f.mcp(w, in)
			return
		}
		if req.URL.Path == "/api/graphql" {
			f.graph(w, in)
			return
		}
		f.rest(w, req, in)
		return
	}
	parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	if len(parts) >= 2 && strings.HasSuffix(parts[1], ".git") {
		f.git(w, req, parts)
		return
	}
	writeJSON(w, 404, object{"message": "Not Found"})
}

func (f *forge) rest(w http.ResponseWriter, req *http.Request, in object) {
	path := strings.Trim(req.URL.Path, "/")
	if path == "api/v3/" || path == "api/v3" {
		w.Header().Set("X-OAuth-Scopes", "repo, read:org")
		writeJSON(w, 200, object{"current_user_url": "/api/v3/user"})
		return
	}
	if path == "api/v3/meta" {
		writeJSON(w, 200, object{"installed_version": "3.0.0"})
		return
	}
	owner := "adaiversen"
	if path == "api/v3/user/repos" && req.Method == "GET" {
		f.mu.RLock()
		out := make([]object, 0, len(f.repos))
		for _, r := range f.repos {
			if r.Owner == owner {
				out = append(out, r.json())
			}
		}
		f.mu.RUnlock()
		writeJSON(w, 200, out)
		return
	}
	if path == "api/v3/user/repos" && req.Method == "POST" {
		if n := text(in["name"]); n != "" {
			if text(in["owner"]) != "" {
				owner = text(in["owner"])
			}
			if r, ok := f.create(owner, n); ok {
				r.Description = text(in["description"])
				writeJSON(w, 201, r.json())
				return
			}
		}
		writeJSON(w, 422, object{"message": "Invalid repository name"})
		return
	}
	parts := strings.Split(path, "/")
	if len(parts) < 5 || parts[0] != "api" || parts[1] != "v3" || parts[2] != "repos" {
		writeJSON(w, 404, object{"message": "Not Found"})
		return
	}
	owner, name := parts[3], parts[4]
	r := f.find(owner, name)
	if r == nil {
		writeJSON(w, 404, object{"message": "Not Found"})
		return
	}
	if len(parts) == 5 {
		switch req.Method {
		case "GET":
			writeJSON(w, 200, r.json())
		case "PATCH":
			r.mu.Lock()
			r.Description = text(in["description"])
			r.mu.Unlock()
			writeJSON(w, 200, r.json())
		case "DELETE":
			f.mu.Lock()
			delete(f.repos, owner+"/"+name)
			f.mu.Unlock()
			writeJSON(w, 204, nil)
		default:
			writeJSON(w, 404, object{"message": "Not Found"})
		}
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	resource := parts[5]
	switch resource {
	case "issues":
		if len(parts) == 6 && req.Method == "POST" {
			for _, raw := range asSlice(in["labels"]) {
				label := text(raw)
				if label != "" && !r.Labels[label] {
					writeJSON(w, 422, object{"message": "Label does not exist"})
					return
				}
			}
			n := len(r.Issues) + 1
			it := issue{Number: n, ID: r.ID*100000 + n, Title: text(in["title"]), Body: text(in["body"]), State: "open"}
			r.Issues = append(r.Issues, it)
			writeJSON(w, 201, it)
			return
		}
		if len(parts) == 6 && req.Method == "GET" {
			writeJSON(w, 200, r.Issues)
			return
		}
		if len(parts) == 7 && req.Method == "GET" {
			n, _ := strconv.Atoi(parts[6])
			for _, it := range r.Issues {
				if it.Number == n {
					writeJSON(w, 200, it)
					return
				}
			}
		}
	case "labels":
		if req.Method == "POST" {
			label := text(in["name"])
			if label != "" {
				r.Labels[label] = true
				writeJSON(w, 201, object{"name": label})
				return
			}
		}
	case "pulls":
		if len(parts) == 6 && req.Method == "POST" {
			n := len(r.PRs) + 1
			pr := object{"id": r.ID*10000 + n, "number": n, "title": text(in["title"]), "body": text(in["body"]), "state": "open", "head": object{"ref": text(in["head"])}, "base": object{"ref": text(in["base"])}, "merged": false}
			r.PRs = append(r.PRs, pr)
			writeJSON(w, 201, pr)
			return
		}
		if len(parts) >= 7 {
			n, _ := strconv.Atoi(parts[6])
			for i, pr := range r.PRs {
				if intValue(pr["number"]) == n {
					if req.Method == "GET" && len(parts) == 7 {
						writeJSON(w, 200, pr)
						return
					}
					if req.Method == "PUT" && len(parts) == 8 && parts[7] == "merge" {
						if text(pr["state"]) != "open" {
							writeJSON(w, 405, object{"message": "Pull request cannot be merged"})
							return
						}
						sha := sha1.Sum([]byte(fmt.Sprint(pr["id"], pr["title"])))
						encoded := hex.EncodeToString(sha[:])
						pr["state"] = "closed"
						pr["merged"] = true
						r.PRs[i] = pr
						r.Refs["refs/heads/main"] = encoded
						writeJSON(w, 200, object{"sha": encoded, "merged": true, "message": "Pull Request successfully merged"})
						return
					}
				}
			}
		}
	}
	writeJSON(w, 404, object{"message": "Not Found"})
}

func asSlice(v any) []any { a, _ := v.([]any); return a }
func intValue(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	}
	return 0
}

func (f *forge) graph(w http.ResponseWriter, in object) {
	q := text(in["query"])
	if strings.Contains(q, "__type") {
		name := "Issue"
		if strings.Contains(q, "PullRequest") {
			name = "PullRequest"
		}
		if strings.Contains(q, "Repository") {
			name = "Repository"
		}
		fields := []object{}
		for _, field := range []string{"id", "number", "title", "state", "body", "headRefName", "baseRefName", "mergeable", "merged", "files", "databaseId", "name", "nameWithOwner", "owner", "defaultBranchRef", "viewerPermission", "hasIssuesEnabled", "issue", "issues", "pullRequests", "pullRequest"} {
			fields = append(fields, object{"name": field})
		}
		writeJSON(w, 200, object{"data": object{"__type": object{"name": name, "fields": fields}, "__schema": object{"types": []object{object{"name": "SearchType", "enumValues": []object{object{"name": "ISSUE"}}}}}}})
		return
	}
	writeJSON(w, 200, object{"data": object{}})
}

func (f *forge) mcp(w http.ResponseWriter, in object) {
	id := in["id"]
	method := text(in["method"])
	switch method {
	case "initialize", "server/discover":
		writeJSON(w, 200, object{"jsonrpc": "2.0", "id": id, "result": object{"protocolVersion": "2026-07-28", "supportedVersions": []string{"2026-07-28", "2025-11-25"}, "capabilities": object{"tools": object{}}, "serverInfo": object{"name": "r5-forge", "version": "1"}}})
	case "notifications/initialized":
		w.WriteHeader(202)
	case "tools/list":
		tools := []object{}
		for _, name := range []string{"create_repo", "get_repo", "create_issue", "list_issues", "create_pull_request", "merge_pull_request"} {
			tools = append(tools, object{"name": name, "description": "Operate on the shared forge store", "inputSchema": object{"type": "object", "properties": object{}}})
		}
		writeJSON(w, 200, object{"jsonrpc": "2.0", "id": id, "result": object{"tools": tools}})
	case "tools/call":
		writeJSON(w, 200, object{"jsonrpc": "2.0", "id": id, "result": object{"content": []object{object{"type": "text", "text": "ok"}}, "isError": false}})
	default:
		writeJSON(w, 200, object{"jsonrpc": "2.0", "id": id, "error": object{"code": -32601, "message": "Method not found"}})
	}
}

func (f *forge) git(w http.ResponseWriter, req *http.Request, parts []string) {
	name := strings.TrimSuffix(parts[1], ".git")
	r := f.find(parts[0], name)
	if r == nil {
		writeJSON(w, 404, object{"message": "Not Found"})
		return
	}
	if strings.Contains(req.URL.Path, "info/refs") {
		service := req.URL.Query().Get("service")
		if service != "git-upload-pack" && service != "git-receive-pack" {
			writeJSON(w, 403, object{"message": "Bad service"})
			return
		}
		w.Header().Set("Content-Type", "application/x-"+service+"-advertisement")
		prefix := "# service=" + service + "\n"
		w.Write([]byte("001e" + prefix + "0000"))
		if service == "git-upload-pack" {
			r.mu.RLock()
			for ref, sha := range r.Refs {
				line := sha + " " + ref + "\x00report-status side-band-64k ofs-delta\n"
				fmt.Fprintf(w, "%04x%s", len(line)+4, line)
			}
			r.mu.RUnlock()
		}
		w.Write([]byte("0000"))
		return
	}
	if strings.HasSuffix(req.URL.Path, "git-upload-pack") {
		w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
		w.Write([]byte("0000"))
		return
	}
	if strings.HasSuffix(req.URL.Path, "git-receive-pack") {
		w.Header().Set("Content-Type", "application/x-git-receive-pack-result")
		w.Write([]byte("00000000"))
		return
	}
	writeJSON(w, 404, object{"message": "Not Found"})
}

func main() {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	server := &http.Server{Handler: newForge(), ReadHeaderTimeout: 5 * time.Second}
	go server.Serve(listener)
	env := strings.TrimRight(os.Getenv("MAKEFUN_ENV_URL"), "/")
	if env == "" {
		fmt.Println("forge listening", listener.Addr())
		select {}
	}
	client := &http.Client{Timeout: 10 * time.Second}
	payload, _ := json.Marshal(object{"port": listener.Addr().(*net.TCPAddr).Port})
	for {
		resp, e := client.Post(env+"/register", "application/json", strings.NewReader(string(payload)))
		if e == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 || resp.StatusCode == 409 {
				break
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	for {
		resp, e := client.Get(env + "/done")
		if e == nil {
			var state object
			_ = json.NewDecoder(resp.Body).Decode(&state)
			resp.Body.Close()
			if state["done"] == true {
				break
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	_ = server.Close()
}
