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
	collection *SourceMap

	count int64
	mu    sync.RWMutex // for count
}

func NewCountPlugin(ctx context.Context, e *EventService) *CountPlugin {
	p := &CountPlugin{collection: NewSourceMap(), count: 0}
	go func() {
		for {
			select {
			default:
			case <-ctx.Done():
				fmt.Println("app done.")
				return
			}
			p.mu.Lock()
			p.count++
			e.Broadcast(sse.Event{Event: "timecount", Data: strconv.FormatInt(p.count, 10)})
			p.mu.Unlock()
			time.Sleep(time.Second)
		}
	}()
	return p
}

func (p *CountPlugin) OnConnected(s *Source) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	s, exists := p.collection.LoadWithKey(s.Key)
	if exists {
		s.Send(sse.Event{Event: "timecount", Data: strconv.FormatInt(p.count, 10)})
	}
}

func (p *CountPlugin) GetTimeCount(ctx context.Context, e *EventService) gin.HandlerFunc {
	return func(c *gin.Context) {
		s, exists := e.Source(c)
		if !exists {
			c.Status(http.StatusBadRequest)
			return
		}

		if c.Query("enable") == "off" {
			p.collection.Delete(c)
			return
		}

		if _, exists := p.collection.Load(c); !exists {
			p.collection.Store(c, s)
		}

		p.mu.RLock()
		defer p.mu.RUnlock()
		s.Send(sse.Event{Event: "timecount", Data: strconv.FormatInt(p.count, 10)})
		c.Status(http.StatusOK)
	}
}
