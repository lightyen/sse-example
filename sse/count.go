package sse

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"
)

type CountPlugin struct {
	e          *EventService
	collection *SourceMap

	count int64
	mu    sync.RWMutex // for count
}

func Count(ctx context.Context, e *EventService) *CountPlugin {
	p := &CountPlugin{e: e, collection: NewSourceMap(), count: 0}
	go p.run(ctx)
	return p
}

func (p *CountPlugin) run(ctx context.Context) {
	for {
		select {
		default:
		case <-ctx.Done():
			fmt.Println("app done.")
			return
		}
		p.mu.Lock()
		p.count++
		p.e.Broadcast(sse.Event{Event: "timecount", Data: strconv.FormatInt(p.count, 10)})
		p.mu.Unlock()
		time.Sleep(time.Second)
	}
}

func (p *CountPlugin) OnSourceConnected(s *Source) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	s, exists := p.collection.LoadWithKey(s.Key)
	if exists {
		s.Send(sse.Event{Event: "timecount", Data: strconv.FormatInt(p.count, 10)})
	}
}

// GET apis/timecount
func (p *CountPlugin) GetTimeCount() gin.HandlerFunc {
	return func(c *gin.Context) {
		s, exists := p.e.LoadWithGinContext(c)
		if !exists {
			c.Status(http.StatusBadRequest)
			return
		}

		if c.Query("enable") == "off" {
			p.collection.Delete(s)
			return
		}

		if _, exists := p.collection.LoadWithGinContext(c); !exists {
			p.collection.Store(s)
		}

		p.mu.RLock()
		defer p.mu.RUnlock()
		s.Send(sse.Event{Event: "timecount", Data: strconv.FormatInt(p.count, 10)})
		c.Status(http.StatusOK)
	}
}
