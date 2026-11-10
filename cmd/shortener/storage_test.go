package main

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStorageSurvivesRestart(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "shortener")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if output, err := exec.CommandContext(ctx, "go", "build", "-buildvcs=false", "-o", binaryPath, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, output)
	}
	for _, mode := range []string{"default", "flag", "environment", "environment overrides flag"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "shortener.json")
			var args, env []string
			switch mode {
			case "flag":
				path = filepath.Join(dir, "flag.json")
				args = []string{"-f", path}
			case "environment":
				path = filepath.Join(dir, "env.json")
				env = []string{"FILE_STORAGE_PATH=" + path}
			case "environment overrides flag":
				path = filepath.Join(dir, "env.json")
				args = []string{"-f", filepath.Join(dir, "unused.json")}
				env = []string{"FILE_STORAGE_PATH=" + path}
			}
			address, stop := startStorageServer(t, binaryPath, dir, args, env)
			first := createStoredLink(t, address, "https://example.com/first", false)
			firstCounter := linkCounter(t, first)
			stop()
			// Explicitly reopen the expected file, proving which path was selected.
			address, stop = startStorageServer(t, binaryPath, dir, []string{"-f", path}, nil)
			assertStoredLink(t, address, first, "https://example.com/first")
			second := createStoredLink(t, address, "https://example.com/second", true)
			if got := linkCounter(t, second); got <= firstCounter {
				t.Fatalf("counter did not advance: %d <= %d", got, firstCounter)
			}
			stop()
			address, stop = startStorageServer(t, binaryPath, dir, []string{"-f", path}, nil)
			defer stop()
			assertStoredLink(t, address, first, "https://example.com/first")
			assertStoredLink(t, address, second, "https://example.com/second")
		})
	}
}

func startStorageServer(t *testing.T, binaryPath, dir string, args, env []string) (string, func()) {
	t.Helper()
	address := freeAddress(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	cmd := exec.CommandContext(ctx, binaryPath, append([]string{"-a", address, "-b", "http://" + address}, args...)...)
	cmd.Dir = dir
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, "SERVER_ADDRESS=") && !strings.HasPrefix(item, "BASE_URL=") && !strings.HasPrefix(item, "FILE_STORAGE_PATH=") {
			cmd.Env = append(cmd.Env, item)
		}
	}
	cmd.Env = append(cmd.Env, env...)
	logFile, err := os.CreateTemp(dir, "server-*.log")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	cmd.Stdout, cmd.Stderr = logFile, logFile
	if err := cmd.Start(); err != nil {
		cancel()
		logFile.Close()
		t.Fatal(err)
	}
	done := make(chan struct{})
	var waitErr error
	go func() { waitErr = cmd.Wait(); close(done) }()
	var once sync.Once
	stop := func() { once.Do(func() { cancel(); <-done; logFile.Close() }) }
	t.Cleanup(stop)
	client := &http.Client{Timeout: time.Second}
	for {
		select {
		case <-done:
			logs, _ := os.ReadFile(logFile.Name())
			t.Fatalf("server exited: %v\n%s", waitErr, logs)
		default:
		}
		response, err := client.Get("http://" + address + "/")
		if err == nil {
			response.Body.Close()
			return address, stop
		}
		select {
		case <-ctx.Done():
			t.Fatalf("server not ready: %v", err)
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func createStoredLink(t *testing.T, address, original string, asJSON bool) string {
	t.Helper()
	endpoint, contentType, body := "/", "text/plain", original
	if asJSON {
		endpoint = "/api/shorten"
		contentType = "application/json"
		data, _ := json.Marshal(map[string]string{"url": original})
		body = string(data)
	}
	client := &http.Client{Timeout: time.Second}
	response, err := client.Post("http://"+address+endpoint, contentType, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d body=%q", response.StatusCode, data)
	}
	if asJSON {
		var result struct {
			Result string `json:"result"`
		}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		return result.Result
	}
	return string(data)
}

func assertStoredLink(t *testing.T, address, short, original string) {
	t.Helper()
	parsed, err := url.Parse(short)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Get("http://" + address + parsed.Path)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusTemporaryRedirect || response.Header.Get("Location") != original {
		t.Fatalf("resolve status=%d location=%q", response.StatusCode, response.Header.Get("Location"))
	}
}

func linkCounter(t *testing.T, short string) uint64 {
	t.Helper()
	parsed, err := url.Parse(short)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.TrimPrefix(parsed.Path, "/")
	if len(id) < 6 {
		t.Fatalf("invalid ID %q", id)
	}
	data, err := base64.RawURLEncoding.DecodeString(id[4:])
	if err != nil || len(data) > 8 {
		t.Fatalf("invalid counter %q", id)
	}
	var padded [8]byte
	copy(padded[8-len(data):], data)
	return binary.BigEndian.Uint64(padded[:])
}
