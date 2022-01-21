package sse

import (
	"bufio"
	"bytes"
	"container/ring"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
	"github.com/google/shlex"
)

// https://docs.microsoft.com/zh-tw/windows/console/console-virtual-terminal-sequences

type TerminalRequest struct {
	Input string `json:"input"`
}

type TerminalPlugin struct {
	name string
}

func Terminal() *TerminalPlugin {
	return &TerminalPlugin{
		name: "terminal",
	}
}

func (p *TerminalPlugin) Name() string { return p.name }

func (p *TerminalPlugin) Setup(srv *EventService, e *gin.RouterGroup) (func(peer *Peer) PeerRunner, func(s *Source) SourceRunner) {
	e.POST("/terminal", func(c *gin.Context) {
		req := &TerminalRequest{}
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
			if t, ok := r.(*TerminalInstance); ok {
				header := func() string {
					errorHeader := ""
					if t.lastExitCode != 0 {
						errorHeader = "\x1b[40m\x1b[0;31m✘ "
					}
					host := "helloworld"
					return "\r\x1b[K\x1b[40m\x1b[1;32m" + errorHeader + "\x1b[40m\x1b[1;32m" + host + "\x1b[0m ~ $\x1b[0m "
				}

				if req.Input == "" {
					t.lastExitCode = 0
					bs, _ := json.Marshal(TerminalResponse{"eof", header()})
					s.Send(sse.Event{Event: "terminal", Data: string(bs)})
				} else {
					t.data <- req.Input
				}

			}
		}

		c.Status(http.StatusOK)
	})

	e.POST("/terminal/cancel", func(c *gin.Context) {
		s := srv.Source(c)
		if s == nil {
			c.Status(http.StatusBadRequest)
			return
		}

		if r, ok := s.plugins[p.name]; ok {
			if t, ok := r.(*TerminalInstance); ok {
				t.Cancel()
			}
		}

		c.Status(http.StatusOK)
	})

	return func(peer *Peer) PeerRunner {
			return nil
		}, func(source *Source) SourceRunner {
			return &TerminalInstance{
				data:   make(chan string, 1),
				cancel: make(chan context.CancelFunc, 1),
			}
		}
}

func (p *TerminalPlugin) Start(ctx context.Context) {}

type TerminalInstance struct {
	data         chan string
	cancel       chan context.CancelFunc
	lastExitCode int
}

type TerminalResponse struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func (t *TerminalInstance) Cancel() {
	for {
		select {
		case cancel := <-t.cancel:
			cancel()
		default:
			return
		}
	}
}

func (t *TerminalInstance) Stop(s *Source) {
	t.Cancel()
}

func (t *TerminalInstance) Start(ctx context.Context, s *Source) {
	r := NewInputReader(ctx, t.data)
	go t.readLoop(ctx, r, s)
}

func (t *TerminalInstance) readLoop(ctx context.Context, r *InputReader, s *Source) {
	buf := bufio.NewReader(r)
	history := ring.New(16)

	printable := func(c rune) bool {
		return c >= 0x20 && c <= 0x7E
	}

	header := func() string {
		errorHeader := ""
		if t.lastExitCode != 0 {
			errorHeader = "\x1b[40m\x1b[0;31m✘ "
		}
		host := "helloworld"
		return "\r\x1b[K\x1b[40m\x1b[1;32m" + errorHeader + "\x1b[40m\x1b[1;32m" + host + "\x1b[0m ~ $\x1b[0m "
	}

	clear := func() bool {
		bs, _ := json.Marshal(TerminalResponse{"clear", ""})
		return s.Send(sse.Event{Event: "terminal", Data: string(bs)})
	}

	send := func(data string) bool {
		bs, _ := json.Marshal(TerminalResponse{"out", data})
		return s.Send(sse.Event{Event: "terminal", Data: string(bs)})
	}

	eof := func(exitCode int) bool {
		t.lastExitCode = exitCode
		bs, _ := json.Marshal(TerminalResponse{"eof", header()})
		return s.Send(sse.Event{Event: "terminal", Data: string(bs)})
	}

	deny := func(filename string) bool {
		t.lastExitCode = 1
		bs, _ := json.Marshal(TerminalResponse{"eof", "command not found: " + filename + "\r\n" + header()})
		return s.Send(sse.Event{Event: "terminal", Data: string(bs)})
	}

	remove := func(s string, i int) string {
		arr := []rune(s)
		if i < len(arr) {
			a := append(arr[:i], arr[i+1:]...)
			return string(a)
		}
		return s
	}

	insert := func(s string, i int, r rune) string {
		arr := []rune(s)
		if i >= len(arr) {
			return s + string(r)
		}
		a := make([]rune, i, len(arr)+1)
		copy(a, arr)
		a = append(a, r)
		a = append(a, arr[i:]...)
		return string(a)
	}

	var myHistory *ring.Ring

SKIP:
	for {
		cursor := 0
		input := ""
	EXEC:
		for {
			c, _, err := buf.ReadRune()
			if err != nil {
				return
			}
			// TODO: handle vt100 code
		READ_CHAR:
			switch c {
			case '\x1b':
				key, last, err := t.readEscape(buf)
				if err != nil {
					return
				}
				switch key {
				case Up:
					if myHistory == nil {
						myHistory = history
					}
					if myHistory != nil {
						val, _ := myHistory.Value.(string)
						if val != "" {
							if len(input) > 0 {
								send("\x1b[" + strconv.FormatInt(int64(cursor), 10) + "D\x1b[K")
							}
							send(val)
							input = val
							cursor = len(val)

							if prev := myHistory.Prev(); prev != nil {
								if _, ok := prev.Value.(string); ok {
									myHistory = prev
								}
							}
						}
					}
				case Down:
					if myHistory != nil {
						var hasNext bool
						if next := myHistory.Next(); next != nil {
							if _, hasNext = next.Value.(string); hasNext {
								myHistory = next
							}
						}

						if hasNext {
							val := myHistory.Value.(string)
							if len(input) > 0 {
								send("\x1b[" + strconv.FormatInt(int64(cursor), 10) + "D\x1b[K")
							}
							send(val)
							input = val
							cursor = len(val)
						}
					}
				case Right:
					cursor++
					send("\x1b[C")
				case Left:
					if cursor > 0 {
						send("\x1b[D")
						cursor--
					}
				case DEL:
					send("\x1b[3~")
				case Unknown:
					c = last
					goto READ_CHAR
				}
			case '\b':
				if cursor > 0 {
					send("\b\x1b[P")
					cursor--
					input = remove(input, cursor)
				}
			case '\x03':
				send("\r\n")
				eof(1)
			case '\r':
				break EXEC
			case '\x0c':
				clear()
			default:
				if printable(c) {
					myHistory = nil
					if cursor < len(input) {
						send("\x1b[1@")
					}
					send(string(c))
					input = insert(input, cursor, c)
					cursor++

				}
			}
		}

		if input != "" {
			val, ok := history.Value.(string)
			if ok && val != input || !ok {
				history = history.Next()
				history.Value = input
			}
		}

		myHistory = nil
		lexer := shlex.NewLexer(strings.NewReader(input))
		words := []string{}
		for val, err := lexer.Next(); err == nil; val, err = lexer.Next() {
			words = append(words, val)
		}
		if len(words) == 0 {
			send("\r\n")
			eof(t.lastExitCode)
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
				send("\r\n")
				_ = deny(words[0])
				continue SKIP
			}
		}

		filename := words[0]

		switch filename {
		case "ping":
		case "traceroute":
		case "nslookup":
		case "date":
		case "clear":
		default:
			send("\r\n")
			deny(filename)
			continue
		}

		if _, err := exec.LookPath(filename); err != nil {
			send("\r\n")
			deny(filename)
			continue
		}

		t.Cancel()
		ctx, cancel := context.WithCancel(ctx)
		t.cancel <- cancel
		cmd := exec.CommandContext(ctx, filename, words[1:]...)
		cmd.Env = os.Environ()
		cmd.Dir, _ = os.UserHomeDir()

		stdin, err := cmd.StdinPipe()
		if err != nil {
			send("\r\n")
			eof(1)
			continue
		}

		go func() {
			defer stdin.Close()
			for {
				select {
				case <-ctx.Done(): // process exit
					return
				case c := <-r.ch:
					if c == 0x03 {
						cancel()
					}
					buf := &bytes.Buffer{}
					if _, err := buf.WriteRune(c); err != nil {
						continue
					}
					if _, err := io.Copy(stdin, buf); err != nil {
						if err == io.EOF {
							return
						}
					}
				}
			}
		}()

		send("\r\n")

		exitCode := t.exec(cmd, func() {
			cancel()
		}, send)

		fmt.Println(exitCode)

		eof(exitCode)
	}
}

type InputReader struct {
	ctx context.Context
	ch  <-chan rune
}

func NewInputReader(ctx context.Context, data <-chan string) *InputReader {
	ch := make(chan rune, 1)

	go func() {
		for s := range data {
			for _, c := range s {
				select {
				case <-ctx.Done():
					return
				case ch <- c:
				}
			}

		}
	}()

	return &InputReader{ctx, ch}
}

func (r *InputReader) Read(p []byte) (n int, err error) {
	select {
	case <-r.ctx.Done():
		return 0, io.EOF
	case s := <-r.ch:
		buffer := &bytes.Buffer{}
		_, _ = buffer.WriteRune(s)
		return buffer.Read(p)
	}
}

func (t *TerminalInstance) exec(cmd *exec.Cmd, exit func(), send func(data string) bool) int {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		send(err.Error() + "\n")
		return 1
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		send(err.Error() + "\n")
		return 1
	}

	if err = cmd.Start(); err != nil {
		send(err.Error() + "\n")
		return 1
	}

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 256)
		for {
			n, err := stdout.Read(buf)
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

	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 256)
		for {
			n, err := stderr.Read(buf)
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

	wg.Wait()

	exit()

	var e *exec.ExitError
	if err := cmd.Wait(); err != nil && !errors.As(err, &e) {
		send(err.Error() + "\n")
	}

	return cmd.ProcessState.ExitCode()
}

type EscapeKey int

const (
	Unknown = iota
	ESC
	DEL // ESC[3~
	Tab // \x09
	CR  // \x0D
	Up
	Down
	Right
	Left
)

var ErrUnknownEscape = errors.New("unknown escape code")

func (t *TerminalInstance) readEscape(buf io.RuneReader) (EscapeKey, rune, error) {
	lastRune, _, err := buf.ReadRune()
	if err != nil {
		return Unknown, lastRune, err
	}

	if lastRune != '[' {
		return ESC, lastRune, nil
	}

	lastRune, _, err = buf.ReadRune()
	if err != nil {
		return Unknown, lastRune, err
	}

	switch lastRune {
	case 'A':
		return Up, lastRune, nil
	case 'B':
		return Down, lastRune, nil
	case 'C':
		return Right, lastRune, nil
	case 'D':
		return Left, lastRune, nil
	case '3':
		lastRune, _, err = buf.ReadRune()
		if err != nil {
			return Unknown, lastRune, err
		}
		switch lastRune {
		case '~':
			return DEL, lastRune, nil
		}
	}

	return Unknown, lastRune, nil
}
