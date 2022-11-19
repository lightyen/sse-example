package sse

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
)

type TemplatePlugin struct {
	name       string
	srv        *EventService
	collection *EventSourceMap
}

type TemplatePluginInstance struct {
	collection *EventSourceMap
}

var (
	_ Plugin         = new(TemplatePlugin)
	_ PluginInstance = new(TemplatePluginInstance)
)

// Show the example that how to use plugin
func Template() *TemplatePlugin {
	return &TemplatePlugin{
		name:       "template", // unique name
		collection: NewSourceMap(),
	}
}

func (p *TemplatePlugin) Name() string { return p.name }

func (p *TemplatePlugin) Install(srv *EventService, e *gin.RouterGroup) func(s *EventSource) PluginInstance {
	// NOTE: setup your routes
	p.srv = srv
	e.GET("/template", func(c *gin.Context) {
		now := time.Now()
		c.String(http.StatusOK, now.String())

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

		if _, ok := p.collection.Load(s.id); !ok {
			p.collection.Store(s.id, s)
		}

		s.Send(&sse.Event{Event: "template", Data: now.String()})
	})
	// NOTE: return instance creators
	return func(source *EventSource) PluginInstance {
		return &TemplatePluginInstance{collection: p.collection}
	}
}
func (p *TemplatePlugin) Run(c context.Context) {
	// NOTE: start running service
	for {
		time.Sleep(time.Second)
		now := time.Now()
		p.srv.Broadcast(&sse.Event{Event: "template", Data: now.String()})
	}
}
func (p *TemplatePlugin) Broadcast(e *sse.Event) {
	// NOTE: broadcast event to group
	p.collection.Range(func(k string, v *EventSource) bool {
		select {
		case <-v.Done():
			p.collection.Delete(k)
		default:
			go v.Send(e)
		}
		return true
	})
}

func (t *TemplatePluginInstance) Run(s *EventSource) {}

func (t *TemplatePluginInstance) Dispose(s *EventSource) {
	// NOTE: remove source from group when connection is closed
	t.collection.Delete(s.id)
}
