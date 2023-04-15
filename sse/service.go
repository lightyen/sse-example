package sse

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
	"reflect"
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
	context.Context
	cancel context.CancelFunc
}

func (s *Source) Send(e Event) {
	c, cancel := context.WithTimeout(s.Context, 10*time.Second)
	defer cancel()
	select {
	case <-c.Done():
		s.cancel()
		return
	case s.Ch <- e:
		return
	}
}

func (s *Source) Close() {
	s.cancel()
}

type SourceMap struct {
	*sync.RWMutex
	m map[string]*Source
}

func NewSourceMap() *SourceMap {
	return &SourceMap{RWMutex: new(sync.RWMutex), m: make(map[string]*Source)}
}

func (m *SourceMap) Load(c *gin.Context) (src *Source, exists bool) {
	m.RLock()
	defer m.RUnlock()
	src, exists = m.m[SourceKeyFunc(c)]
	return
}

func (m *SourceMap) Store(c *gin.Context, v *Source) {
	m.Lock()
	defer m.Unlock()
	m.m[SourceKeyFunc(c)] = v
}

func (m *SourceMap) Delete(c *gin.Context) {
	m.Lock()
	defer m.Unlock()
	delete(m.m, SourceKeyFunc(c))
}

func (m *SourceMap) Range(callback func(s *Source) bool) {
	m.RLock()
	defer m.RUnlock()
	for _, v := range m.m {
		if callback(v) == false {
			break
		}
	}
}

func (m *SourceMap) RangeAndDelete(cb func(v *Source) bool) {
	m.Lock()
	defer m.Unlock()
	for k, v := range m.m {
		delete(m.m, k)
		if cb(v) == false {
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

// Register register global source initializer
func (s *EventService) Register(args ...SourceOnStartup) {
	s.onstartupMutex.Lock()
	defer s.onstartupMutex.Unlock()
	s.onstartup = append(s.onstartup, args...)
}

func (s *EventService) Unregister(args ...SourceOnStartup) {
	s.onstartupMutex.Lock()
	defer s.onstartupMutex.Unlock()

	var onstartup []reflect.Value
	var targets []reflect.Value
	for i := 0; i < len(s.onstartup); i++ {
		onstartup = append(onstartup, reflect.ValueOf(s.onstartup[i]))
	}
	for j := 0; j < len(args); j++ {
		targets = append(targets, reflect.ValueOf(args[j]))
	}

	var result []SourceOnStartup
	for i := 0; i < len(onstartup); i++ {
		var exists = false
		for j := 0; j < len(targets); j++ {
			if onstartup[i].Pointer() == targets[j].Pointer() {
				exists = true
				break
			}
		}
		if !exists {
			result = append(result, s.onstartup[i])
		}
	}
	s.onstartup = result
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
	s.sources.Range(func(v *Source) bool {
		v.Send(e)
		return true
	})
}

func (s *EventService) CloseAll() {
	s.sources.RangeAndDelete(func(v *Source) bool {
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

func New() *EventService {
	return &EventService{sources: NewSourceMap()}
}

func (s *EventService) Range(cb func(s *Source) bool) {
	s.sources.Range(cb)
}

func (s *EventService) GinStreamHandler(ctx *gin.Context) {
	if !strings.Contains(ctx.Request.Header.Get("Accept"), "text/event-stream") {
		ctx.Status(400)
		return
	}

	sourceKey := SourceKeyFunc(ctx)
	if sourceKey == "" {
		sourceKey = hex.EncodeToString(randomId())
	}

	ch := make(chan Event, 1)
	c, cancel := context.WithCancel(ctx)
	defer cancel()

	source := &Source{
		ID:        sourceKey,
		Ch:        ch,
		Context:   c,
		cancel:    cancel,
		ClientIP:  ctx.ClientIP(),
		UserAgent: ctx.GetHeader("User-Agent"),
	}

	s.sources.Lock()
	s.sources.m[sourceKey] = source
	s.sources.Unlock()

	s.onstartupMutex.RLock()
	for i := range s.onstartup {
		go s.onstartup[i](source)
	}
	s.onstartupMutex.RUnlock()

	go func() {
		w := ctx.Writer
		header := w.Header()
		header.Set("Cache-Control", "no-store")
		header.Set("Content-Type", "text/event-stream")
		header.Set("Connection", "keep-alive")
		ctx.Render(200, &Event{Event: "establish", Retry: 3000, Id: sourceKey, Data: sourceKey})
		w.Flush()

		for {
			select {
			case <-c.Done():
				return
			case evt := <-ch:
				ctx.Render(200, evt)
				w.Flush()
			}
		}
	}()

	<-ctx.Writer.CloseNotify()
	cancel()
	s.sources.Lock()
	defer s.sources.Unlock()
	delete(s.sources.m, sourceKey)
}
