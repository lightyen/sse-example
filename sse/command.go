package sse

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
	"github.com/google/shlex"
)

type SingleCommandRequest struct {
	Input string `json:"input"`
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

		s := srv.Source(c)
		if s == nil {
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
SKIP:
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-t.request:
			t.Cancel()
			ctx, cancel := context.WithCancel(ctx)
			t.cancel <- cancel

			type Response struct {
				Type string `json:"type"`
				Data string `json:"data"`
			}

			header := "\r\x1b[K\x1b[40m\x1b[32m helloworld \x1b[30m\x1b[104m\x1b[30m root \x1b[40m\x1b[94m\x1b[0m "
			send := func(data string) bool {
				bs, _ := json.Marshal(Response{"data", data})
				return s.Send(sse.Event{Event: "command", Data: string(bs)})
			}
			eof := func() bool {
				bs, _ := json.Marshal(Response{"eof", header})
				return s.Send(sse.Event{Event: "command", Data: string(bs)})
			}
			deny := func() bool {
				bs, _ := json.Marshal(Response{"eof", "Access Deny\r\n" + header})
				return s.Send(sse.Event{Event: "command", Data: string(bs)})
			}

			// TODO 1: parse input
			lexer := shlex.NewLexer(strings.NewReader(req.Input))
			words := []string{}
			for val, err := lexer.Next(); err == nil; val, err = lexer.Next() {
				words = append(words, val)
			}
			if len(words) == 0 {
				eof()
				continue
			}

			for i := range words {
				switch {
				case strings.HasPrefix(words[i], "|"):
					fallthrough
				case strings.HasPrefix(words[i], "&"):
					fallthrough
				case strings.HasPrefix(words[i], ">"):
					fallthrough
				case strings.HasPrefix(words[i], "<"):
					deny()
					continue SKIP
				}
			}

			// TODO 2: whitelist
			switch words[0] {
			case "ping":
			case "traceroute":
			case "tracert":
			case "nslookup":
			case "echo":
			default:
				deny()
				continue
			}

			go t.runCommand(ctx, words[0], words[1:], send, eof)
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

func (t *SingleCommandInstance) runCommand(ctx context.Context, name string, args []string, send func(data string) bool, eof func() bool) {
	if _, err := exec.LookPath(name); err != nil {
		send(err.Error() + "\n")
		eof()
		return
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stderr = cmd.Stdout
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return
	}

	defer func() {
		var e *exec.ExitError
		if err := cmd.Wait(); err != nil && !errors.As(err, &e) {
			send(err.Error() + "\n")
		}
		eof()
	}()

	if err = cmd.Start(); err != nil {
		send(err.Error() + "\n")
		return
	}
	buf := make([]byte, 256)
	for {
		n, err := cmdReader.Read(buf)
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
