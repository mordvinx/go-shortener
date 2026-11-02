package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestEnvironmentPriority(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "shortener")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if output, err := exec.CommandContext(ctx, "go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build server: %v\n%s", err, output)
	}

	for _, tt := range []struct {
		name                    string
		flagAddress, envAddress bool
		flagBase, envBase       string
		emptyEnv                bool
		wantBase                string
	}{
		{name: "defaults", wantBase: "http://localhost:8080/"},
		{name: "flags only", flagAddress: true, flagBase: "http://flags.example", wantBase: "http://flags.example/"},
		{name: "environment only", envAddress: true, envBase: "http://env.example", wantBase: "http://env.example/"},
		{name: "environment overrides flags", flagAddress: true, envAddress: true, flagBase: "http://flags.example", envBase: "http://env.example", wantBase: "http://env.example/"},
		{name: "address from environment and base from flag", envAddress: true, flagBase: "http://flags.example", wantBase: "http://flags.example/"},
		{name: "address from flag and base from environment", flagAddress: true, envBase: "http://env.example", wantBase: "http://env.example/"},
		{name: "default base with custom address", flagAddress: true, wantBase: "http://localhost:8080/"},
		{name: "empty environment preserves flags", flagAddress: true, flagBase: "http://flags.example", emptyEnv: true, wantBase: "http://flags.example/"},
		{name: "empty environment preserves defaults", emptyEnv: true, wantBase: "http://localhost:8080/"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			address := "127.0.0.1:8080"
			var args, env []string
			if tt.flagAddress {
				address = freeAddress(t)
				args = append(args, "-a", address)
			}
			if tt.flagBase != "" {
				args = append(args, "-b", tt.flagBase)
			}
			if tt.envAddress {
				// A distinct address makes ignoring SERVER_ADDRESS observable.
				envAddress := freeAddress(t)
				for envAddress == address {
					envAddress = freeAddress(t)
				}
				address = envAddress
				env = append(env, "SERVER_ADDRESS="+address)
			}
			if tt.envBase != "" {
				env = append(env, "BASE_URL="+tt.envBase)
			}
			if tt.emptyEnv {
				env = append(env, "SERVER_ADDRESS=", "BASE_URL=")
			}
			if !tt.flagAddress && !tt.envAddress {
				listener, err := net.Listen("tcp4", "0.0.0.0:8080")
				if errors.Is(err, syscall.EADDRINUSE) {
					t.Skipf("default port 8080 is occupied: %v", err)
				}
				if err != nil {
					t.Fatal(err)
				}
				listener.Close()
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, args...)
			for _, item := range os.Environ() {
				if !strings.HasPrefix(item, "SERVER_ADDRESS=") && !strings.HasPrefix(item, "BASE_URL=") {
					cmd.Env = append(cmd.Env, item)
				}
			}
			cmd.Env = append(cmd.Env, env...)
			logPath := filepath.Join(t.TempDir(), "server.log")
			logFile, err := os.Create(logPath)
			if err != nil {
				t.Fatal(err)
			}
			cmd.Stdout, cmd.Stderr = logFile, logFile
			if err := cmd.Start(); err != nil {
				logFile.Close()
				t.Fatal(err)
			}
			done := make(chan struct{})
			var waitErr error
			go func() { waitErr = cmd.Wait(); close(done) }()
			t.Cleanup(func() {
				cancel()
				<-done
				logFile.Close()
			})

			transport := &http.Transport{Proxy: nil}
			defer transport.CloseIdleConnections()
			client := &http.Client{
				Transport:     transport,
				Timeout:       time.Second,
				CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
			}
			const original = "https://practicum.yandex.ru/"
			var response *http.Response
			for {
				select {
				case <-done:
					output, _ := os.ReadFile(logPath)
					t.Fatalf("server exited: %v\n%s", waitErr, output)
				default:
				}
				response, err = client.Post("http://"+address+"/", "text/plain", strings.NewReader(original))
				if err == nil {
					break
				}
				select {
				case <-ctx.Done():
					t.Fatalf("server not ready at %s: %v", address, err)
				case <-time.After(20 * time.Millisecond):
				}
			}
			body, err := io.ReadAll(response.Body)
			response.Body.Close()
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != http.StatusCreated {
				t.Fatalf("POST status %d, body %q", response.StatusCode, body)
			}
			shortURL := string(body)
			if !strings.HasPrefix(shortURL, tt.wantBase) || len(shortURL) == len(tt.wantBase) {
				t.Fatalf("short URL %q must start with %q and contain an ID", shortURL, tt.wantBase)
			}
			id := strings.TrimPrefix(shortURL, tt.wantBase)
			resolved, err := client.Get("http://" + address + "/" + id)
			if err != nil {
				t.Fatal(err)
			}
			resolved.Body.Close()
			if resolved.StatusCode != http.StatusTemporaryRedirect || resolved.Header.Get("Location") != original {
				t.Errorf("GET status %d, Location %q", resolved.StatusCode, resolved.Header.Get("Location"))
			}
		})
	}
}

func freeAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	return address
}
