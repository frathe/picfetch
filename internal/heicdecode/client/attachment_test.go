package client

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestInheritedRemoteUsesOwner(t *testing.T) {
	runInheritedRemote(t, "success", "--owned-heic-remote", "owned remote decode complete\n")
}

func TestInheritedRemoteCancellationJoins(t *testing.T) {
	runInheritedRemote(t, "hang", "--owned-heic-remote-cancel", "owned remote cancellation complete\n")
}

func runInheritedRemote(t *testing.T, peer, mode, want string) {
	t.Helper()
	owner := ownedPeer(t, peer, 5*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0])
	link, err := owner.Attach(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { link.Stop(); link.Wait() }()
	config, err := json.Marshal(link.Config)
	if err != nil {
		t.Fatal(err)
	}
	cmd.Args = append(cmd.Args, mode, string(config))
	var output, diagnostic bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &diagnostic
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	link.Started()
	if err = cmd.Wait(); err != nil {
		t.Fatalf("inherited remote: %v (%s)", err, diagnostic.String())
	}
	if output.String() != want {
		t.Fatalf("unexpected child output: %q", output.String())
	}
}

func TestAttachmentStopWithoutProcessStartJoinsOwner(t *testing.T) {
	owner := ownedPeer(t, "success", 5*time.Second)
	link, err := owner.Attach(context.Background(), exec.Command(os.Args[0]))
	if err != nil {
		t.Fatal(err)
	}
	link.Stop()
	link.Wait()
	link.Started()
	link.Stop()
	owner.Stop()
	owner.Wait()
}
