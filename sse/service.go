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
	Key       string
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
	mu *sync.RWMutex
	m  map[string]*Source
}

func NewSourceMap() *SourceMap {
	return &SourceMap{mu: new(sync.RWMutex), m: make(map[string]*Source)}
}

func (m *SourceMap) LoadWithKey(key string) (src *Source, exists bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	src, exists = m.m[key]
	return
}

func (m *SourceMap) StoreWithKey(key string, v *Source) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[key] = v
}

func (m *SourceMap) DeleteWithKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, key)
}

func (m *SourceMap) Range(callback func(*Source) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, v := range m.m {
		if callback(v) == false {
			break
		}
	}
}

func (m *SourceMap) RangeAndDelete(cb func(*Source) bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range m.m {
		delete(m.m, k)
		if cb(v) == false {
			break
		}
	}
}

// Load with gin.Context
func (m *SourceMap) Load(c *gin.Context) (src *Source, exists bool) {
	return m.LoadWithKey(SourceKeyFunc(c))
}

// Store with gin.Context
func (m *SourceMap) Store(c *gin.Context, v *Source) {
	m.StoreWithKey(SourceKeyFunc(c), v)
}

// Delete with gin.Context
func (m *SourceMap) Delete(c *gin.Context) {
	m.DeleteWithKey(SourceKeyFunc(c))
}

type Event = sse.Event

type OnSourceConnected func(s *Source)

type EventService struct {
	sources          *SourceMap
	onconnedtedMutex sync.RWMutex
	onconnected      []OnSourceConnected
}

var DefaultSourceKeyFunc = func(c *gin.Context) string {
	return c.GetHeader("Last-Event-ID")
}

var SourceKeyFunc = DefaultSourceKeyFunc

// Register register global source initializer
func (s *EventService) Register(args ...OnSourceConnected) {
	s.onconnedtedMutex.Lock()
	defer s.onconnedtedMutex.Unlock()
	s.onconnected = append(s.onconnected, args...)
}

func (s *EventService) Unregister(args ...OnSourceConnected) {
	s.onconnedtedMutex.Lock()
	defer s.onconnedtedMutex.Unlock()

	var onstartup []reflect.Value
	var targets []reflect.Value
	for i := 0; i < len(s.onconnected); i++ {
		onstartup = append(onstartup, reflect.ValueOf(s.onconnected[i]))
	}
	for j := 0; j < len(args); j++ {
		targets = append(targets, reflect.ValueOf(args[j]))
	}

	var result []OnSourceConnected
	for i := 0; i < len(onstartup); i++ {
		var exists = false
		for j := 0; j < len(targets); j++ {
			if onstartup[i].Pointer() == targets[j].Pointer() {
				exists = true
				break
			}
		}
		if !exists {
			result = append(result, s.onconnected[i])
		}
	}
	s.onconnected = result
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

// GET /apis/stream
func (s *EventService) StreamHandlerFunc(c *gin.Context) {
	if !strings.Contains(c.Request.Header.Get("Accept"), "text/event-stream") {
		c.Status(400)
		return
	}

	sourceKey := SourceKeyFunc(c)
	if sourceKey == "" {
		sourceKey = hex.EncodeToString(randomId())
	}

	ch := make(chan Event, 1)
	ctx, cancel := context.WithCancel(c)
	defer cancel()

	source := &Source{
		Key:       sourceKey,
		Ch:        ch,
		Context:   ctx,
		cancel:    cancel,
		ClientIP:  c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}

	s.sources.StoreWithKey(sourceKey, source)

	s.onconnedtedMutex.RLock()
	for i := range s.onconnected {
		go s.onconnected[i](source)
	}
	s.onconnedtedMutex.RUnlock()

	go func() {
		w := c.Writer
		header := w.Header()
		header.Set("Cache-Control", "no-store")
		header.Set("Content-Type", "text/event-stream")
		header.Set("Connection", "keep-alive")
		c.Render(200, &Event{Event: "establish", Retry: 3000, Id: sourceKey, Data: sourceKey})
		w.Flush()

		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-ch:
				c.Render(200, evt)
				w.Flush()
			}
		}
	}()

	<-c.Writer.CloseNotify()
	cancel()
	s.sources.DeleteWithKey(sourceKey)
}
