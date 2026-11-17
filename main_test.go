package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func call(t *testing.T, f *forge, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer forge-demo")
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	f.ServeHTTP(res, req)
	return res
}

func TestRepositoryAndIssuesShareStore(t *testing.T) {
	f := newForge()
	created := call(t, f, http.MethodPost, "/api/v3/user/repos", object{"name": "demo"})
	if created.Code != http.StatusCreated {
		t.Fatalf("create repository status=%d body=%s", created.Code, created.Body)
	}
	issueRes := call(t, f, http.MethodPost, "/api/v3/repos/adaiversen/demo/issues", object{"title": "Bug", "body": "details"})
	if issueRes.Code != http.StatusCreated {
		t.Fatalf("create issue status=%d", issueRes.Code)
	}
	read := call(t, f, http.MethodGet, "/api/v3/repos/adaiversen/demo/issues/1", nil)
	if read.Code != http.StatusOK {
		t.Fatalf("read issue status=%d", read.Code)
	}
}

func TestAuthenticationAndUnknownResource(t *testing.T) {
	f := newForge()
	req := httptest.NewRequest(http.MethodGet, "/api/v3/user/repos", nil)
	res := httptest.NewRecorder()
	f.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d", res.Code)
	}
	missing := call(t, f, http.MethodGet, "/api/v3/repos/adaiversen/nope", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing status=%d", missing.Code)
	}
}
