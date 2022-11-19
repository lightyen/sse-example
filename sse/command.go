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
		name: "single_command",
	}
}

type SingleCommandInstance struct {
	request chan *SingleCommandRequest
	cancel  chan context.CancelFunc
}

var (
	_ Plugin         = new(SingleCommandPlugin)
	_ PluginInstance = new(SingleCommandInstance)
)

func (p *SingleCommandPlugin) Name() string { return p.name }

func (p *SingleCommandPlugin) Install(srv *EventService, e *gin.RouterGroup) func(s *EventSource) PluginInstance {
	e.POST("/command", func(c *gin.Context) {
		req := &SingleCommandRequest{}
		if err := c.ShouldBindJSON(req); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		switch req.Name {
		case "seq":
		default:
			c.Status(http.StatusBadRequest)
			return
		}

		s, exists := srv.FromContext(c)
		if !exists {
			c.Status(http.StatusBadRequest)
			return
		}

		for _, r := range s.pluginInstances {
			switch t := r.(type) {
			case *SingleCommandInstance:
				t.request <- req
				break
			}
		}
		c.Status(http.StatusOK)
	})

	e.POST("/command/cancel", func(c *gin.Context) {
		s, exists := srv.FromContext(c)
		if !exists {
			c.Status(http.StatusBadRequest)
			return
		}

		for _, r := range s.pluginInstances {
			switch t := r.(type) {
			case *SingleCommandInstance:
				t.CancelCommand()
				break
			}
		}

		c.Status(http.StatusOK)
	})

	return func(source *EventSource) PluginInstance {
		return &SingleCommandInstance{
			request: make(chan *SingleCommandRequest, 1),
			cancel:  make(chan context.CancelFunc, 1),
		}
	}
}

func (p *SingleCommandPlugin) Run(ctx context.Context) {}

func (t *SingleCommandInstance) Run(s *EventSource) {
	var eventID int64 = 0
	for {
		select {
		case <-s.Done():
			return
		case req := <-t.request:
			eventID++
			t.CancelCommand()
			ctx, cancel := context.WithCancel(s)
			t.cancel <- cancel
			send := func(data string) {
				// TODO: move id into data
				s.Send(&sse.Event{Id: strconv.FormatInt(eventID, 10), Event: "command", Data: data})
			}
			go t.runCommand(ctx, req.Name, req.Args, send)
		}
	}
}

func (t *SingleCommandInstance) CancelCommand() {
	for {
		select {
		case cancel := <-t.cancel:
			cancel()
		default:
			return
		}
	}
}

func (t *SingleCommandInstance) Dispose(s *EventSource) {
	t.CancelCommand()
}

func (t *SingleCommandInstance) runCommand(c context.Context, name string, args []string, send func(data string)) {
	cmd := exec.CommandContext(c, name, args...)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	cmd.Stderr = cmd.Stdout
	if err = cmd.Start(); err != nil {
		send(err.Error())
		return
	}
	defer func() {
		var e *exec.ExitError
		if err := cmd.Wait(); err != nil && !errors.As(err, &e) {
			send(err.Error())
		}
	}()
	buf := make([]byte, 256)
	for {
		n, err := cmdReader.Read(buf)
		if n > 0 {
			send(strings.Replace(string(buf[:n]), "\r\n", "\n", -1))
		}
		if err != nil {
			if err != io.EOF {
				send(err.Error())
			}
			return
		}
	}
}
