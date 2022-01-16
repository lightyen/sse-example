package sse

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
)

type ShellRequest struct {
	Input string `json:"input"`
}

type ShellPlugin struct {
	name string
}

func Shell() *ShellPlugin {
	return &ShellPlugin{
		name: "shell",
	}
}

func (p *ShellPlugin) Name() string { return p.name }

func (p *ShellPlugin) Setup(srv *EventService, e *gin.RouterGroup) (func(p *Peer) PeerRunner, func(s *Source) SourceRunner) {
	e.POST("/shell", func(c *gin.Context) {
		req := &ShellRequest{}
		if err := c.ShouldBindJSON(req); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		s := srv.Source(c)
		if s == nil {
			c.Status(http.StatusBadRequest)
			return
		}

		if r, ok := s.plugins[p.name]; ok {
			if t, ok := r.(*ShellInstance); ok {
				t.request <- req
			}
		}

		c.Status(http.StatusOK)
	})

	return func(peer *Peer) PeerRunner {
			return nil
		}, func(source *Source) SourceRunner {
			return &ShellInstance{
				request: make(chan *ShellRequest, 1),
				cancel:  make(chan context.CancelFunc, 1),
			}
		}
}

func (p *ShellPlugin) Serve(ctx context.Context) {}

type ShellInstance struct {
	request chan *ShellRequest
	cancel  chan context.CancelFunc
}

func (t *ShellInstance) Run(ctx context.Context, s *Source) {
	var cmd *exec.Cmd
	var w io.WriteCloser

	for {
		select {
		case <-ctx.Done():
			return
		case req := <-t.request:
			if cmd == nil {
				filename := "zsh"
				if runtime.GOOS == "windows" {
					filename = "pwsh"
				}
				ctx, cancel := context.WithCancel(ctx)
				cmd = exec.CommandContext(ctx, filename)
				cmd.Stderr = cmd.Stdout
				w, _ = cmd.StdinPipe()
				t.cancel <- cancel
				type Response struct {
					Type string `json:"type"`
					Data string `json:"data"`
				}
				send := func(data string) bool {
					bs, _ := json.Marshal(Response{"data", data})
					return s.Send(sse.Event{Event: "shell", Data: string(bs)})
				}
				go t.runCommand(cmd, send)
				continue
			}

			w.Write([]byte(req.Input))
		}
	}
}

func (t *ShellInstance) Cancel() {
	for {
		select {
		case cancel := <-t.cancel:
			cancel()
		default:
			return
		}
	}
}

func (t *ShellInstance) Stop(s *Source) {
	t.Cancel()
}

func (t *ShellInstance) runCommand(cmd *exec.Cmd, send func(data string) bool) {
	cmdStdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}

	cmdStderr, err := cmd.StderrPipe()
	if err != nil {
		return
	}

	defer func() {
		var e *exec.ExitError
		if err := cmd.Wait(); err != nil && !errors.As(err, &e) {
			send(err.Error() + "\n")
		}
	}()

	if err = cmd.Start(); err != nil {
		send(err.Error() + "\n")
		return
	}

	go func() {
		buf := make([]byte, 256)
		for {
			n, err := cmdStderr.Read(buf)
			if n > 0 {
				send(string(buf[:n]))
			}
			if err != nil {
				if err != io.EOF {
					send(err.Error() + "\n")
				}
				return
			}
		}
	}()

	buf := make([]byte, 256)
	for {
		n, err := cmdStdout.Read(buf)
		if n > 0 {
			send(string(buf[:n]))
		}
		if err != nil {
			if err != io.EOF {
				send(err.Error() + "\n")
			}
			return
		}
	}
}
