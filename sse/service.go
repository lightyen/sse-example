package sse

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
)

type Sender interface {
	Send(e Event)
	Close()
}

type Source struct {
	ID        string
	Ch        chan Event
	ClientIP  string
	UserAgent string
	Cancel    context.CancelFunc
	context.Context
}

func (s *Source) Send(e Event) {
	c, cancel := context.WithTimeout(s.Context, 10*time.Second)
	defer cancel()
	defer func() {
		if c.Err() == context.DeadlineExceeded {
			// client bug
			s.Cancel()

		}
	}()
	select {
	case <-c.Done():
		return
	case s.Ch <- e:
		return
	}
}

func (s *Source) Close() {
	s.Cancel()
}

type SourceMap struct {
	m  map[string]*Source
	mu *sync.RWMutex
}

func NewSourceMap() *SourceMap {
	return &SourceMap{m: make(map[string]*Source), mu: new(sync.RWMutex)}
}

func (s *SourceMap) Load(c *gin.Context) (src *Source, exists bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	src, exists = s.m[SourceKeyFunc(c)]
	return
}

func (s *SourceMap) Store(c *gin.Context, v *Source) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[SourceKeyFunc(c)] = v
}

func (s *SourceMap) Delete(c *gin.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, SourceKeyFunc(c))
}

func (s *SourceMap) Range(callback func(k string, v *Source) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, v := range s.m {
		if callback(k, v) == false {
			break
		}
	}
}

func (s *SourceMap) RangeAndDelete(callback func(k string, v *Source) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.m {
		delete(s.m, k)
		if callback(k, v) == false {
			break
		}
	}
}

type Event = sse.Event
type SourceOnStartup func(s *Source)

type EventService struct {
	sources        *SourceMap
	onstartup      []SourceOnStartup
	onstartupMutex sync.RWMutex
}

var DefaultSourceKeyFunc = func(c *gin.Context) string {
	return c.GetHeader("Last-Event-ID")
}

var SourceKeyFunc = DefaultSourceKeyFunc

func (s *EventService) Register(args ...SourceOnStartup) {
	s.onstartupMutex.Lock()
	defer s.onstartupMutex.Unlock()
	s.onstartup = append(s.onstartup, args...)
}

func (s *EventService) Source(c *gin.Context) (src *Source, exists bool) {
	return s.sources.Load(c)
}

func (s *EventService) Send(c *gin.Context, e Event) {
	src, ok := s.Source(c)
	if ok {
		src.Send(e)
	}
}

func (s *EventService) Broadcast(e Event) {
	s.sources.Range(func(k string, v *Source) bool {
		v.Send(e)
		return true
	})
}

func (s *EventService) CloseAll() {
	s.sources.RangeAndDelete(func(k string, v *Source) bool {
		v.Close()
		return true
	})
}

func randomUint16() uint16 {
	var b [2]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(err)
	}
	return binary.BigEndian.Uint16(b[:])
}

func randomUint32() uint32 {
	var b [4]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(err)
	}
	return binary.BigEndian.Uint32(b[:])
}

var counter = randomUint32()
var pid = uint16(os.Getpid())

func randomId() []byte {
	var b [16]byte
	binary.BigEndian.PutUint32(b[0:], atomic.AddUint32(&counter, 1))
	binary.BigEndian.PutUint16(b[4:], randomUint16())
	binary.BigEndian.PutUint64(b[6:], uint64(time.Now().Unix()))
	binary.BigEndian.PutUint16(b[14:], pid)
	return b[:]
}

func NewEventSourceService() *EventService {
	return &EventService{sources: NewSourceMap()}
}

func (s *EventService) StreamHandlerFunc(c *gin.Context) {
	if !strings.Contains(c.Request.Header.Get("Accept"), "text/event-stream") {
		c.Status(http.StatusBadRequest)
		return
	}

	sourceKey := SourceKeyFunc(c)
	if sourceKey == "" {
		sourceKey = hex.EncodeToString(randomId())
	}

	ch := make(chan Event, 1)
	ctx, cancel := context.WithCancel(c)
	defer cancel()

	go func() {
		<-c.Writer.CloseNotify()
		s.sources.mu.Lock()
		delete(s.sources.m, sourceKey)
		s.sources.mu.Unlock()
		cancel()
	}()

	source := &Source{
		ID:        sourceKey,
		Ch:        ch,
		Context:   ctx,
		Cancel:    cancel,
		ClientIP:  c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}

	s.sources.mu.Lock()
	s.sources.m[sourceKey] = source
	s.sources.mu.Unlock()

	s.onstartupMutex.RLock()
	for i := range s.onstartup {
		go s.onstartup[i](source)
	}
	s.onstartupMutex.RUnlock()

	w := c.Writer
	header := w.Header()
	header.Set("Cache-Control", "no-store")
	header.Set("Content-Type", "text/event-stream")
	header.Set("Connection", "keep-alive")
	c.Render(http.StatusOK, &Event{Event: "establish", Retry: 3000, Id: sourceKey, Data: sourceKey})
	w.Flush()

	for {
		select {
		case <-ctx.Done():
			return
		case e := <-ch:
			c.Render(http.StatusOK, e)
			// NOTE: webpack-dev-server would not close the connection when the client is closed.
			w.Flush()
		}
	}
}

type SourceInfo struct {
	ID        interface{} `json:"id"`
	ClientIP  string      `json:"ip"`
	UserAgent string      `json:"ua"`
}

func (s *EventService) Sources() []*Source {
	var items []*Source
	s.sources.Range(func(k string, s *Source) bool {
		items = append(items, s)
		return true
	})
	return items
}
