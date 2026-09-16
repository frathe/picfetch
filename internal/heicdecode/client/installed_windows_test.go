//go:build windows && (amd64 || arm64)

package client

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/heicdecode"
)

func TestWindowsInstalledHelperCache(t *testing.T) {
	source := filepath.Join(t.TempDir(), "owned.exe")
	data := []byte("owned fixture, never executed")
	if err := os.WriteFile(source, data, 0700); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	root := filepath.Join(t.TempDir(), "private-cache")
	staged, release, err := prepareInstalled(context.Background(), source, digest, root)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if staged == source || !strings.Contains(staged, hex.EncodeToString(digest[:])) {
		t.Fatal("installed source was not privately staged")
	}
	first, err := os.Stat(staged)
	if err != nil {
		t.Fatal(err)
	}
	reused, release2, err := prepareInstalled(context.Background(), source, digest, root)
	if err != nil {
		t.Fatal(err)
	}
	defer release2()
	info, err := os.Stat(reused)
	if err != nil || !os.SameFile(first, info) {
		t.Fatal("valid active helper copy was replaced")
	}
	if err = os.WriteFile(staged, []byte("damaged"), 0700); err == nil {
		t.Fatal("active executable was writable")
	}
	if err = os.Remove(staged); err == nil {
		t.Fatal("active executable was deletable")
	}
	release()
	release2()
	if err = os.WriteFile(staged, []byte("damaged"), 0700); err != nil {
		t.Fatal(err)
	}
	repaired, release3, err := prepareInstalled(context.Background(), source, digest, root)
	if err != nil {
		t.Fatal(err)
	}
	defer release3()
	got, err := os.ReadFile(repaired)
	if err != nil || string(got) != string(data) {
		t.Fatal("damaged cache was not repaired")
	}
	newer := []byte("owned newer helper")
	if err = os.WriteFile(source, newer, 0700); err != nil {
		t.Fatal(err)
	}
	current, release4, err := prepareInstalled(context.Background(), source, sha256.Sum256(newer), root)
	if err != nil {
		t.Fatal(err)
	}
	defer release4()
	if _, err = os.Stat(repaired); err != nil {
		t.Fatal("removed obsolete helper while leased")
	}
	release3()
	again, release5, err := prepareInstalled(context.Background(), source, sha256.Sum256(newer), root)
	if err != nil {
		t.Fatal(err)
	}
	defer release5()
	if again != current {
		t.Fatal("current helper path changed")
	}
	if _, err = os.Stat(repaired); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unused obsolete copy survived cleanup: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err = prepareInstalled(ctx, source, sha256.Sum256(newer), root); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled stage: %v", err)
	}
}

func TestWindowsCacheLeaseWaitsForShutdown(t *testing.T) {
	source := filepath.Join(t.TempDir(), "owned.exe")
	data := []byte("owned helper")
	if err := os.WriteFile(source, data, 0700); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	path, release, err := prepareInstalled(context.Background(), source, digest, filepath.Join(t.TempDir(), "cache"))
	if err != nil {
		t.Fatal(err)
	}
	owner, err := New(Config{Executable: path, SHA256: digest, Limits: heicdecode.DefaultLimits(0)})
	if err != nil {
		release()
		t.Fatal(err)
	}
	owner.release = release
	defer func() { owner.Stop(); owner.Wait() }()
	owner.Wait()
	if err = os.Remove(path); err == nil {
		t.Fatal("nonterminal Wait released the helper lease")
	}
	owner.Stop()
	owner.Wait()
	owner.Wait()
	if err = os.Remove(path); err != nil {
		t.Fatalf("shutdown retained the helper lease: %v", err)
	}
}

func TestWindowsConcurrentHelperStaging(t *testing.T) {
	source := filepath.Join(t.TempDir(), "owned.exe")
	data := []byte("owned simultaneous publication")
	if err := os.WriteFile(source, data, 0700); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "cache")
	config, err := json.Marshal([]string{source, root})
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	type peer struct {
		command *exec.Cmd
		input   io.WriteCloser
		output  *bufio.Reader
	}
	var peers []peer
	for range 3 {
		cmd := exec.CommandContext(ctx, executable, "-test.run=^TestWindowsHelperCachePeer$")
		cmd.Env = append(os.Environ(), "PICFETCH_HEIC_CACHE_TEST_PEER="+string(config))
		input, pipeErr := cmd.StdinPipe()
		if pipeErr != nil {
			t.Fatal(pipeErr)
		}
		output, pipeErr := cmd.StdoutPipe()
		if pipeErr != nil {
			t.Fatal(pipeErr)
		}
		cmd.Stderr = os.Stderr
		if err = cmd.Start(); err != nil {
			t.Fatal(err)
		}
		peers = append(peers, peer{cmd, input, bufio.NewReader(output)})
	}
	defer func() {
		for _, peer := range peers {
			_ = peer.input.Close()
			_ = peer.command.Wait()
		}
	}()
	var selected string
	for _, peer := range peers {
		line, readErr := peer.output.ReadString('\n')
		if readErr != nil || !strings.HasPrefix(line, "ready ") {
			t.Fatalf("cache peer: %q, %v", line, readErr)
		}
		path := strings.TrimSpace(strings.TrimPrefix(line, "ready "))
		if selected != "" && selected != path {
			t.Fatal("concurrent publishers selected different copies")
		}
		selected = path
	}
	if got, readErr := os.ReadFile(selected); readErr != nil || string(got) != string(data) {
		t.Fatal("concurrent publication exposed incomplete bytes")
	}
	for _, peer := range peers {
		if err = peer.input.Close(); err != nil {
			t.Fatal(err)
		}
		if err = peer.command.Wait(); err != nil {
			t.Fatal(err)
		}
	}
	peers = nil
}

func TestWindowsHelperCachePeer(t *testing.T) {
	value := os.Getenv("PICFETCH_HEIC_CACHE_TEST_PEER")
	if value == "" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(value), &args); err != nil || len(args) != 2 {
		t.Fatal("invalid owned peer configuration")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		t.Fatal(err)
	}
	path, release, err := prepareInstalled(context.Background(), args[0], sha256.Sum256(data), args[1])
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	fmt.Println("ready " + path)
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func TestWindowsHelperCacheRejectsHardLinksAndNonDirectories(t *testing.T) {
	source := filepath.Join(t.TempDir(), "owned.exe")
	data := []byte("owned helper")
	if err := os.WriteFile(source, data, 0700); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	root := filepath.Join(t.TempDir(), "cache")
	path, release, err := prepareInstalled(context.Background(), source, digest, root)
	if err != nil {
		t.Fatal(err)
	}
	release()
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = os.Link(source, path); err != nil {
		t.Fatal(err)
	}
	if _, stop, err := prepareInstalled(context.Background(), source, digest, root); err == nil {
		stop()
		t.Fatal("cache hard link could change installed source permissions")
	}
	if _, stop, err := prepareInstalled(context.Background(), source, digest, source); err == nil {
		stop()
		t.Fatal("file accepted as private cache directory")
	}
}
