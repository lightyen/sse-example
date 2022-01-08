package sse

import (
	"context"
	"sync"

	"github.com/gin-contrib/sse"
)

type Sender interface {
	Send(e sse.Event) bool
	Close()
}

type Source struct {
	key     interface{}
	peer    *Peer
	ch      chan sse.Event
	plugins map[string]SourceRunner
	ctx     context.Context
	cancel  context.CancelFunc
}

func (s *Source) Send(e sse.Event) bool {
	select {
	case <-s.ctx.Done():
		return false
	default:
	}
	s.ch <- e
	return true
}

func (s *Source) Close() {
	s.cancel()
	s.peer.sources.Delete(s.key)
	for _, p := range s.plugins {
		p.Stop(s)
	}
}

type Peer struct {
	key       interface{}
	sources   *sync.Map // <key: string, value: *Source>
	plugins   map[string]PeerRunner
	clientIP  string
	userAgent string
	ctx       context.Context
}

func (p *Peer) Send(e sse.Event) bool {
	var sent bool
	p.sources.Range(func(key, value interface{}) bool {
		if s, ok := value.(*Source); ok {
			if ok := s.Send(e); !ok {
				s.Close()
			} else {
				sent = true
			}
		}
		return true
	})
	return sent
}

func (p *Peer) Close() {
	p.sources.Range(func(key, value interface{}) bool {
		if s, ok := value.(*Source); ok {
			s.Close()
		}
		return true
	})
	for _, plugin := range p.plugins {
		plugin.Stop(p)
	}
}
