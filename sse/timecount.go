package sse

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
)

type TimeCountPlugin struct {
	name       string
	srv        *EventService
	collection *EventSourceMap

	mu    sync.RWMutex
	count int64
}

type TimeCountPluginInstance struct {
	collection *EventSourceMap
}

var (
	_ Plugin         = new(TimeCountPlugin)
	_ PluginInstance = new(TimeCountPluginInstance)
)

// APIs: "/timecount"
func TimeCount() *TimeCountPlugin {
	return &TimeCountPlugin{
		name:       "time_count",
		collection: NewSourceMap(),
	}
}

func (p *TimeCountPlugin) Name() string { return p.name }

func (p *TimeCountPlugin) Install(srv *EventService, e *gin.RouterGroup) func(s *EventSource) PluginInstance {
	p.srv = srv
	e.GET("/timecount", func(c *gin.Context) {
		disabled := c.Query("enable") == "off"

		s, exists := srv.FromContext(c)
		if !exists {
			c.Status(http.StatusBadRequest)
			return
		}

		if disabled {
			p.collection.Delete(s.id)
			return
		}

		if _, exists := p.collection.Load(s.id); !exists {
			p.collection.Store(s.id, s)
		}

		p.mu.RLock()
		defer p.mu.RUnlock()
		s.Send(&sse.Event{Event: "timecount", Data: strconv.FormatInt(p.count, 10)})
		c.Status(http.StatusOK)
	})

	return func(source *EventSource) PluginInstance {
		return &TimeCountPluginInstance{collection: p.collection}
	}
}

func (p *TimeCountPlugin) Run(c context.Context) {
	addOne := func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		p.count++
	}
	for {
		time.Sleep(time.Second)

		select {
		case <-c.Done():
			return
		default:
		}

		addOne()

		p.Broadcast(&sse.Event{Event: "timecount", Data: strconv.FormatInt(p.count, 10)})
	}
}

func (p *TimeCountPlugin) Broadcast(e *sse.Event) {
	p.collection.Range(func(k string, v *EventSource) bool {
		v.Send(e)
		return true
	})
}

func (t *TimeCountPluginInstance) Run(s *EventSource) {}

func (t *TimeCountPluginInstance) Dispose(s *EventSource) {
	t.collection.Delete(s.id)
}
