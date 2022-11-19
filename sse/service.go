package sse

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
)

type EventService struct {
	getId   func(c *gin.Context) string
	sources *EventSourceMap
}

func (srv *EventService) Send(key string, e *sse.Event) {
	if v, exist := srv.sources.Load(key); exist {
		v.Send(e)
	}
}

func (srv *EventService) Broadcast(e *sse.Event) {
	srv.sources.Range(func(k string, v *EventSource) bool {
		v.Send(e)
		return true
	})
}

func (srv *EventService) FromContext(c *gin.Context) (*EventSource, bool) {
	return srv.sources.Load(srv.getId(c))
}

func (srv *EventService) CloseAll() {
	srv.sources.RangeAndDelete(func(k string, v *EventSource) bool {
		v.Close()
		return true
	})
}

func randomId() string {
	var buf [16]byte
	_, err := io.ReadFull(rand.Reader, buf[:])
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf[:])
}

func NewEventService(ctx context.Context, g *gin.RouterGroup, plugins ...Plugin) *EventService {
	getId := func(c *gin.Context) string {
		return c.GetHeader("Last-Event-ID")
	}

	srv := &EventService{getId: getId, sources: NewSourceMap()}

	createHandler := func(builders []func(s *EventSource) PluginInstance) gin.HandlerFunc {
		return func(c *gin.Context) {
			if !strings.Contains(c.Request.Header.Get("Accept"), "text/event-stream") {
				c.Status(http.StatusBadRequest)
				return
			}

			id := getId(c)
			if id == "" {
				id = randomId()
			}

			ch := make(chan *sse.Event, 2)
			cc, cancel := context.WithCancel(c)
			defer cancel()
			defer srv.sources.Delete(id)

			go func() {
				<-c.Writer.CloseNotify()
				cancel()
			}()

			for {
				if _, exists := srv.sources.Load(id); !exists {
					break
				}
				id = randomId()
			}

			s := &EventSource{
				Context:         cc,
				cancel:          cancel,
				id:              id,
				ch:              ch,
				pluginInstances: make([]PluginInstance, 0),
				userAgent:       c.GetHeader("User-Agent"),
				clientIP:        c.ClientIP(),
			}
			defer s.Close()

			for _, h := range builders {
				if obj := h(s); obj != nil {
					s.pluginInstances = append(s.pluginInstances, obj)
					go obj.Run(s)
				}
			}

			srv.sources.Store(id, s)

			w := c.Writer
			header := w.Header()
			header.Set("Cache-Control", "no-store")
			header.Set("Content-Type", "text/event-stream")
			header.Set("Connection", "keep-alive")
			c.Render(http.StatusOK, &sse.Event{Event: "establish", Retry: 3000, Id: id, Data: id})
			w.Flush()

			for {
				select {
				case <-s.Done():
					return
				case e := <-ch:
					c.Render(http.StatusOK, e)
					w.Flush()
				}
			}
		}
	}

	instanceCreators := make([]func(s *EventSource) PluginInstance, 0)
	for _, p := range plugins {
		instanceCreators = append(instanceCreators, p.Install(srv, g))
		go p.Run(ctx)
	}

	g.GET("/stream", createHandler(instanceCreators))
	g.GET("/stream/sources", func(c *gin.Context) {
		c.JSON(http.StatusOK, struct {
			Data interface{} `json:"data"`
		}{srv.Sources()})
	})
	g.GET("/stream/sources/count", func(c *gin.Context) {
		c.JSON(http.StatusOK, struct {
			Data interface{} `json:"data"`
		}{len(srv.Sources())})
	})

	return srv
}

type PeerInfo struct {
	ID        interface{} `json:"id"`
	ClientIP  string      `json:"ip"`
	UserAgent string      `json:"user_agent"`
	Sources   []string    `json:"sources"`
}

type SourceInfo struct {
	ID        interface{} `json:"id"`
	ClientIP  string      `json:"ip"`
	UserAgent string      `json:"ua"`
}

func (s *EventService) Sources() []SourceInfo {
	items := []SourceInfo{}
	s.sources.Range(func(k string, s *EventSource) bool {
		items = append(items, SourceInfo{ID: s.id, ClientIP: s.clientIP, UserAgent: s.userAgent})
		return true
	})
	return items
}
