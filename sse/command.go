package sse

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
)

type SingleCommandRequest struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
}

type SingleCommandPlugin struct {
	name string
}

// APIs: "/command", "/cancel"
func SingleCommand() *SingleCommandPlugin {
	return &SingleCommandPlugin{
		name: "single command",
	}
}

func (p *SingleCommandPlugin) Name() string { return p.name }

func (p *SingleCommandPlugin) Setup(srv *EventService, e *gin.RouterGroup) (func(p *Peer) PeerRunner, func(s *Source) SourceRunner) {
	e.POST("/command", func(c *gin.Context) {
		req := &SingleCommandRequest{}
		if err := c.ShouldBindJSON(req); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		switch req.Name {
		case "ping":
		default:
			c.Status(http.StatusBadRequest)
			return
		}

		s := srv.Source(c)
		if p == nil {
			c.Status(http.StatusBadRequest)
			return
		}

		if r, ok := s.plugins[p.name]; ok {
			if t, ok := r.(*SingleCommandInstance); ok {
				t.request <- req
			}
		}
		c.Status(http.StatusOK)
	})

	e.POST("/cancel", func(c *gin.Context) {
		s := srv.Source(c)
		if s == nil {
			c.Status(http.StatusBadRequest)
			return
		}

		if r, ok := s.plugins[p.name]; ok {
			if t, ok := r.(*SingleCommandInstance); ok {
				t.Cancel()
			}
		}

		c.Status(http.StatusOK)
	})

	return func(peer *Peer) PeerRunner {
			return nil
		}, func(source *Source) SourceRunner {
			return &SingleCommandInstance{
				request: make(chan *SingleCommandRequest, 1),
				cancel:  make(chan context.CancelFunc, 1),
			}
		}
}

func (p *SingleCommandPlugin) Serve(ctx context.Context) {}

type SingleCommandInstance struct {
	request chan *SingleCommandRequest
	cancel  chan context.CancelFunc
}

func (t *SingleCommandInstance) Run(ctx context.Context, s *Source) {
	var eventID int64 = 0
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-t.request:
			eventID++
			t.Cancel()
			ctx, cancel := context.WithCancel(ctx)
			t.cancel <- cancel
			send := func(data string) bool {
				// TODO: move id into data
				return s.Send(sse.Event{Id: strconv.FormatInt(eventID, 10), Event: "command", Data: data})
			}
			go t.runCommand(ctx, req.Name, req.Args, send)
		}
	}
}

func (t *SingleCommandInstance) Cancel() {
	for {
		select {
		case cancel := <-t.cancel:
			cancel()
		default:
			return
		}
	}
}

func (t *SingleCommandInstance) Stop(s *Source) {
	t.Cancel()
}

func (t *SingleCommandInstance) runCommand(ctx context.Context, name string, args []string, send func(data string) bool) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	cmd.Stderr = cmd.Stdout
	if err = cmd.Start(); err != nil {
		_ = send(err.Error())
		return
	}
	defer func() {
		var e *exec.ExitError
		if err := cmd.Wait(); err != nil && !errors.As(err, &e) {
			_ = send(err.Error())
		}
	}()
	buf := make([]byte, 256)
	for {
		n, err := cmdReader.Read(buf)
		if n > 0 {
			_ = send(strings.Replace(string(buf[:n]), "\r\n", "\n", -1))
		}
		if err != nil {
			if err != io.EOF {
				_ = send(err.Error())
			}
			return
		}
	}
}
