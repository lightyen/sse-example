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
	collection *sync.Map

	mu    sync.RWMutex
	count int64
}

// APIs: "/timecount"
func TimeCount() *TimeCountPlugin {
	return &TimeCountPlugin{
		name:       "time count",
		collection: &sync.Map{},
	}
}

func (p *TimeCountPlugin) Name() string { return p.name }

func (p *TimeCountPlugin) Setup(srv *EventService, e *gin.RouterGroup) (func(p *Peer) PeerRunner, func(s *Source) SourceRunner) {
	p.srv = srv
	e.GET("/timecount", func(c *gin.Context) {
		disabled := c.Query("enable") == "off"

		s := srv.Source(c)
		if s == nil {
			c.Status(http.StatusBadRequest)
			return
		}

		if disabled {
			_, _ = p.collection.LoadAndDelete(s.key)
			return
		}

		if _, ok := p.collection.Load(s.key); !ok {
			p.collection.Store(s.key, s)
		}

		p.mu.RLock()
		defer p.mu.RUnlock()
		s.Send(sse.Event{Event: "timecount", Data: strconv.FormatInt(p.count, 10)})
		c.Status(http.StatusOK)
	})

	return func(peer *Peer) PeerRunner {
			return nil
		}, func(source *Source) SourceRunner {
			return &TimeCountPluginInstance{collection: p.collection}
		}
}

func (p *TimeCountPlugin) Serve(c context.Context) {
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

		// push data
		p.Broadcast(sse.Event{Event: "timecount", Data: strconv.FormatInt(p.count, 10)})
	}
}

func (p *TimeCountPlugin) Broadcast(e sse.Event) {
	p.collection.Range(func(key, value interface{}) bool {
		if s, ok := value.(Sender); ok {
			_ = s.Send(e)
		}
		return true
	})
}

type TimeCountPluginInstance struct {
	collection *sync.Map
}

func (t *TimeCountPluginInstance) Run(c context.Context, s *Source) {}
func (t *TimeCountPluginInstance) Stop(s *Source) {
	t.collection.Delete(s.key)
}
