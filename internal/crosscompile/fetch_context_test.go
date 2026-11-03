package crosscompile

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func assertDownloadDirectory(t *testing.T, destination, want string) {
	t.Helper()
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != want {
		t.Fatalf("destination = %q, %v; want %q", data, err, want)
	}
	entries, err := os.ReadDir(filepath.Dir(destination))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(destination) {
		t.Fatalf("download left temporary files: %v", entries)
	}
}

func TestDownloadFileContextCanceledBeforeRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "replacement")
	}))
	defer server.Close()

	for _, expired := range []bool{false, true} {
		name := "canceled"
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		want := context.Canceled
		if expired {
			name = "expired"
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			defer cancel()
			want = context.DeadlineExceeded
		}
		t.Run(name, func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), "sdk.bin")
			if err := os.WriteFile(destination, []byte("previous"), 0600); err != nil {
				t.Fatal(err)
			}
			err := downloadFileContext(ctx, server.Client(), server.URL, destination)
			if !errors.Is(err, want) {
				t.Errorf("download error = %v, want %v", err, want)
			}
			assertDownloadDirectory(t, destination, "previous")
		})
	}
}

func TestDownloadFileContextBodyCancellation(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		name := "cancel"
		if deadline {
			name = "deadline"
		}
		t.Run(name, func(t *testing.T) {
			started := make(chan struct{})
			release := make(chan struct{})
			var releaseOnce sync.Once
			unblock := func() { releaseOnce.Do(func() { close(release) }) }
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "100")
				fmt.Fprint(w, "prefix")
				w.(http.Flusher).Flush()
				close(started)
				select {
				case <-r.Context().Done():
				case <-release:
				}
			}))
			defer server.Close()
			defer unblock()

			ctx, cancel := context.WithCancel(context.Background())
			want := context.Canceled
			if deadline {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
				want = context.DeadlineExceeded
			}
			defer cancel()
			destination := filepath.Join(t.TempDir(), "sdk.bin")
			if err := os.WriteFile(destination, []byte("previous"), 0600); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				done <- downloadFileContext(ctx, server.Client(), server.URL, destination)
			}()
			select {
			case <-started:
			case err := <-done:
				t.Fatalf("download ended before the body started: %v", err)
			case <-time.After(10 * time.Second):
				unblock()
				cancel()
				t.Fatal("local download did not start")
			}
			if !deadline {
				cancel()
			}
			var err error
			select {
			case err = <-done:
			case <-time.After(5 * time.Second):
				// Also drain the old implementation when it ignores cancellation;
				// neither the handler nor its request may survive this failed test.
				unblock()
				select {
				case err = <-done:
				case <-time.After(5 * time.Second):
					t.Fatal("download did not drain after releasing its body")
				}
			}
			if !errors.Is(err, want) {
				t.Errorf("body download error = %v, want %v", err, want)
			}
			assertDownloadDirectory(t, destination, "previous")
		})
	}
}

func TestDownloadFileCommitsOnlyCompleteResponse(t *testing.T) {
	for _, response := range []string{"complete", "not-found", "truncated"} {
		t.Run(response, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch response {
				case "not-found":
					http.NotFound(w, r)
				case "truncated":
					w.Header().Set("Content-Length", "100")
					fmt.Fprint(w, "prefix")
				default:
					fmt.Fprint(w, "complete")
				}
			}))
			defer server.Close()
			for _, existing := range []bool{false, true} {
				name := "new-file"
				if existing {
					name = "replace-file"
				}
				t.Run(name, func(t *testing.T) {
					destination := filepath.Join(t.TempDir(), "sdk.bin")
					if existing {
						if err := os.WriteFile(destination, []byte("previous"), 0600); err != nil {
							t.Fatal(err)
						}
					}
					err := downloadFileContext(context.Background(), server.Client(), server.URL, destination)
					if response == "complete" {
						if err != nil {
							t.Errorf("complete response failed: %v", err)
						}
						assertDownloadDirectory(t, destination, "complete")
						return
					}
					if err == nil {
						t.Error("incomplete response committed without an error")
					}
					if existing {
						assertDownloadDirectory(t, destination, "previous")
					} else {
						entries, err := os.ReadDir(filepath.Dir(destination))
						if err != nil || len(entries) != 0 {
							t.Fatalf("failed download left a destination or temporary file: %v, %v", entries, err)
						}
					}
				})
			}
		})
	}
}

func TestDownloadFailureRemovesOnlyOwnedInstallStaging(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	destination := filepath.Join(t.TempDir(), "esp-clang")
	lockPath := destination + ".lock"
	if err := os.WriteFile(lockPath, nil, 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	err = installESPClang(destination, func(extracted string) error {
		return downloadAndExtractArchive(server.URL+"/sdk.tar.gz", extracted, "local SDK")
	})
	if err == nil {
		t.Fatal("failed SDK download was installed")
	}
	for _, path := range []string{destination, destination + ".extract", destination + ".extract.temp"} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("failed download retained %s: %v", filepath.Base(path), err)
		}
	}
	after, err := os.Stat(lockPath)
	if err != nil || !os.SameFile(before, after) {
		t.Fatalf("installation changed the stable lock file: %v", err)
	}
	lock, err := acquireLock(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := releaseLock(lock); err != nil {
		t.Fatal(err)
	}
}
