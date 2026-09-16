package client

import (
	"context"
	"os"
	"os/exec"
	"sync"

	"github.com/frathe/picfetch/internal/heicdecode"
)

// PipeConfig names only handles explicitly inherited by an analysis process.
// Limits must match its application's owner; no executable path is forwarded.
type PipeConfig struct {
	Input, Output uint64
	Limits        heicdecode.Limits
}

// Attachment owns the parent's service and its temporary copies of child pipe
// ends. Call Started immediately after exec.Start; Stop/Wait on every return.
type Attachment struct {
	Config   PipeConfig
	children []*os.File
	once     sync.Once
	cancel   context.CancelFunc
	done     chan struct{}
}

func (c *Client) Attach(ctx context.Context, cmd *exec.Cmd) (*Attachment, error) {
	if cmd == nil || cmd.Process != nil {
		return nil, ErrUnavailable
	}
	finish, err := c.beginService()
	if err != nil {
		return nil, err
	}
	transferred := false
	var files []*os.File
	defer func() {
		if !transferred {
			for _, file := range files {
				_ = file.Close()
			}
			finish()
		}
	}()
	requestRead, requestWrite, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	files = append(files, requestRead, requestWrite)
	responseRead, responseWrite, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	files = append(files, responseRead, responseWrite)
	readID, writeID, copies, err := inheritBrokerPipes(cmd, responseRead, requestWrite)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	attachment := &Attachment{
		Config:   PipeConfig{Input: readID, Output: writeID, Limits: c.config.Limits},
		children: append([]*os.File{responseRead, requestWrite}, copies...), cancel: cancel, done: make(chan struct{}),
	}
	transferred = true
	go func() {
		defer finish()
		defer close(attachment.done)
		_ = c.serve(ctx, requestRead, responseWrite)
	}()
	return attachment, nil
}

func (a *Attachment) Started() {
	a.once.Do(func() {
		for _, file := range a.children {
			_ = file.Close()
		}
	})
}
func (a *Attachment) Stop() { a.cancel(); a.Started() }
func (a *Attachment) Wait() { <-a.done }

// OpenRemote consumes the two inherited handles. They must be pipes; on UNIX,
// nonblocking mode is installed before NewFile so Close can interrupt reads.
func OpenRemote(config PipeConfig) (*Remote, error) {
	if err := config.Limits.Validate(); err != nil {
		return nil, err
	}
	if config.Input <= 2 || config.Output <= 2 || config.Input == config.Output {
		return nil, ErrUnavailable
	}
	input, err := openBrokerPipe(config.Input, true)
	if err != nil {
		return nil, err
	}
	output, err := openBrokerPipe(config.Output, false)
	if err != nil {
		_ = input.Close()
		return nil, err
	}
	remote, err := NewRemote(input, output, config.Limits)
	if err != nil {
		_ = input.Close()
		_ = output.Close()
		return nil, err
	}
	return remote, nil
}
