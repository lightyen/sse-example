package sse

import (
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EventService struct {
	peerKeyFunc   func(c *gin.Context) string
	sourceKeyFunc func(c *gin.Context) string
	peers         *sync.Map // <key: string, value: *Peer>
}

func (s *EventService) Peer(c *gin.Context) *Peer {
	if v, ok := s.peers.Load(s.peerKeyFunc(c)); ok {
		if peer, ok := v.(*Peer); ok {
			return peer
		}
	}
	return nil
}

func (s *EventService) Source(c *gin.Context) *Source {
	peer := s.Peer(c)
	if peer == nil {
		return nil
	}
	if v, ok := peer.sources.Load(s.sourceKeyFunc(c)); ok {
		if s, ok := v.(*Source); ok {
			return s
		}
	}
	return nil
}

func (s *EventService) Send(key interface{}, e sse.Event) {
	if v, ok := s.peers.Load(key); ok {
		if peer, ok := v.(*Peer); ok {
			if ok := peer.Send(e); !ok {
				s.peers.Delete(key)
			}
		}
	}
}

func (s *EventService) Broadcast(e sse.Event) {
	s.peers.Range(func(key, value interface{}) bool {
		if peer, ok := value.(*Peer); ok {
			if ok := peer.Send(e); !ok {
				s.peers.Delete(key)
			}
		}
		return true
	})
}

func (s *EventService) CloseAll() {
	s.peers.Range(func(key, value interface{}) bool {
		if peer, ok := value.(*Peer); ok {
			s.peers.Delete(key)
			if peer != nil {
				peer.Close()
			}
		}
		return true
	})
}

func NewEventService(ctx context.Context, g *gin.RouterGroup, plugins ...Plugin) *EventService {
	peerKeyFunc := func(c *gin.Context) string {
		return "user-id-1234"
	}
	sourceKeyFunc := func(c *gin.Context) string {
		return c.GetHeader("X-Source-Id")
	}

	s := &EventService{peerKeyFunc: peerKeyFunc, sourceKeyFunc: sourceKeyFunc, peers: &sync.Map{}}

	type PluginBuilder struct {
		peer   func(p *Peer) PeerRunner
		source func(s *Source) SourceRunner
	}

	createHandler := func(builders map[string]PluginBuilder) gin.HandlerFunc {
		return func(c *gin.Context) {
			if !strings.Contains(c.Request.Header.Get("Accept"), "text/event-stream") {
				c.Status(http.StatusBadRequest)
				return
			}

			peerKey := peerKeyFunc(c)
			u := uuid.New()
			sourceKey := u.String()

			ch := make(chan sse.Event, 1)
			ctx, cancel := context.WithCancel(c)

			go func() {
				<-c.Writer.CloseNotify()
				cancel()
			}()

			var peer *Peer
			var source *Source

			defer func() {
				source.Close()
			}()

			NewPeer := func() *Peer {
				peer := &Peer{
					key:       peerKey,
					clientIP:  c.ClientIP(),
					userAgent: c.GetHeader("User-Agent"),
					sources:   &sync.Map{},
					plugins:   map[string]PeerRunner{},
					ctx:       ctx,
				}
				for name, b := range builders {
					if _, exists := peer.plugins[name]; exists {
						panic("plugin name '" + name + "' is conflict")
					}
					if obj := b.peer(peer); obj != nil {
						peer.plugins[name] = obj
						go peer.plugins[name].Run(ctx, peer)
					}
				}
				return peer
			}

			NewSource := func() *Source {
				source := &Source{
					key:     sourceKey,
					peer:    peer,
					ch:      ch,
					plugins: map[string]SourceRunner{},
					ctx:     ctx,
					cancel:  cancel,
				}
				for name, b := range builders {
					if _, exists := source.plugins[name]; exists {
						panic("plugin name '" + name + "' is conflict")
					}
					if obj := b.source(source); obj != nil {
						source.plugins[name] = obj
						go source.plugins[name].Run(ctx, source)
					}
				}
				return source
			}

			if p, ok := s.peers.Load(peerKey); ok {
				peer = p.(*Peer)
			} else {
				peer = NewPeer()
				s.peers.Store(peerKey, peer)
			}

			if _, ok := peer.sources.Load(sourceKey); ok {
				panic("uuid collision")
			}

			source = NewSource()
			peer.sources.Store(sourceKey, source)

			w := c.Writer
			header := w.Header()
			header.Set("Cache-Control", "no-store")
			header.Set("Content-Type", "text/event-stream")
			header.Set("Connection", "keep-alive")
			c.Render(http.StatusOK, sse.Event{Event: "establish", Retry: 3000, Data: sourceKey})
			w.Flush()

			for {
				select {
				case <-ctx.Done():
					return
				case e := <-ch:
					c.Render(http.StatusOK, e)
					w.Flush()
				}
			}
		}
	}

	m := map[string]PluginBuilder{}
	for _, p := range plugins {
		peer, source := p.Setup(s, g)
		m[p.Name()] = PluginBuilder{peer, source}
		go p.Serve(ctx)
	}

	g.GET("", createHandler(m))
	g.GET("/peers", func(c *gin.Context) {
		c.JSON(http.StatusOK, struct {
			Data interface{} `json:"data"`
		}{s.Peers()})
	})
	g.GET("/sources", func(c *gin.Context) {
		c.JSON(http.StatusOK, struct {
			Data interface{} `json:"data"`
		}{s.Sources()})
	})

	return s
}

type PeerInfo struct {
	ID        interface{} `json:"id"`
	ClientIP  string      `json:"ip"`
	UserAgent string      `json:"user_agent"`
	Sources   []string    `json:"sources"`
}

func (s *EventService) Peers() []PeerInfo {
	items := []PeerInfo{}
	sources := []string{}
	s.peers.Range(func(key, value interface{}) bool {
		peer, _ := value.(*Peer)
		peer.sources.Range(func(key, value interface{}) bool {
			source, _ := value.(*Source)
			sources = append(sources, source.key.(string))
			return true
		})
		items = append(items, PeerInfo{ID: peer.key, UserAgent: peer.userAgent, ClientIP: peer.clientIP, Sources: sources})
		return true
	})
	return items
}

type SourceInfo struct {
	ID       interface{} `json:"id"`
	ClientIP string      `json:"ip"`
}

func (s *EventService) Sources() []SourceInfo {
	items := []SourceInfo{}
	s.peers.Range(func(key, value interface{}) bool {
		peer, _ := value.(*Peer)
		peer.sources.Range(func(key, value interface{}) bool {
			source, _ := value.(*Source)
			items = append(items, SourceInfo{ID: source.key, ClientIP: peer.clientIP})
			return true
		})
		return true
	})
	return items
}
