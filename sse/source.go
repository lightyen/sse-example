package sse

import (
	"context"
	"sync"
	"time"

	"github.com/gin-contrib/sse"
)

type EventSource struct {
	context.Context
	cancel          context.CancelFunc
	id              string
	ch              chan *sse.Event
	pluginInstances []PluginInstance
	userAgent       string
	clientIP        string
}

func (s *EventSource) Send(e *sse.Event) {
	c, cancel := context.WithTimeout(s, 10*time.Second)
	defer cancel()
	select {
	case <-c.Done():
	case s.ch <- e:
	}
}

// Cancel self context
func (s *EventSource) Cancel() {
	s.cancel()
}

func (s *EventSource) Close() {
	for _, p := range s.pluginInstances {
		p.Dispose(s)
	}
	s.Cancel()
}

type EventSourceMap struct {
	m  map[string]*EventSource
	mu *sync.RWMutex
}

func NewSourceMap() *EventSourceMap {
	return &EventSourceMap{m: make(map[string]*EventSource), mu: new(sync.RWMutex)}
}

func (s *EventSourceMap) Load(key string) (*EventSource, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	src, exists := s.m[key]
	return src, exists
}

func (s *EventSourceMap) Store(k string, v *EventSource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[k] = v
}

func (s *EventSourceMap) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, id)
}

func (s *EventSourceMap) Range(callback func(k string, v *EventSource) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, v := range s.m {
		if callback(k, v) == false {
			break
		}
	}
}

func (s *EventSourceMap) RangeAndDelete(callback func(k string, v *EventSource) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range s.m {
		delete(s.m, k)
		if callback(k, v) == false {
			break
		}
	}
}
