package similarity

import (
	"context"
	"os/exec"

	heicclient "github.com/frathe/picfetch/internal/heicdecode/client"
	"github.com/frathe/picfetch/internal/imaging"
)

func (c Client) attachHEIC(ctx context.Context, cmd *exec.Cmd, req *request) (*heicclient.Attachment, error) {
	req.HEIC = nil
	if c.HEIC == nil {
		return nil, nil
	}
	link, err := c.HEIC.Attach(ctx, cmd)
	if err != nil {
		return nil, err
	}
	req.HEIC = &link.Config
	return link, nil
}

func closeHEICAttachment(link *heicclient.Attachment) {
	if link != nil {
		link.Stop()
		link.Wait()
	}
}

// Setup and disposal belong to the worker request, before any source work and
// before WorkerMain's exit. A remote has no executable or fallback decoder.
func openWorkerSource(req *request) (func(), error) {
	if req.HEIC == nil {
		return func() {}, nil
	}
	remote, err := heicclient.OpenRemote(*req.HEIC)
	if err != nil {
		return nil, err
	}
	req.reader = imaging.NewReader(remote.Do)
	return func() { remote.Stop(); remote.Wait() }, nil
}
