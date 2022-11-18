package main

import (
	"app/sse"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

func noCacheFirstPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/" {
			c.Header("Cache-Control", "no-store")
		}
	}
}

func fallback(filename string, allowAny bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := c.Request
		if req.Method != "GET" {
			return
		}

		if strings.Contains(c.Request.Header.Get("Accept"), "text/html") {
			c.Header("Cache-Control", "no-store")
			c.File(filename)
			c.Abort()
			return
		}

		if allowAny && strings.Contains(c.Request.Header.Get("Accept"), "*/*") {
			c.Header("Cache-Control", "no-store")
			c.File(filename)
			c.Abort()
			return
		}
	}
}

func run(ctx context.Context, g *gin.Engine) *sse.EventService {
	e := sse.NewEventSourceService()
	g.GET("/apis/stream", e.StreamHandlerFunc)

	e.Register(func(s *sse.Source) {
		cnt := 0
		for {
			select {
			default:
			case <-s.Done():
				fmt.Println("I'm done.")
				return
			}
			time.Sleep(time.Second)
			cnt++
			fmt.Println("my count", cnt)
		}
	})

	// custom feature
	{
		var count int64
		mu := &sync.RWMutex{}
		collection := sse.NewSourceMap()
		go func() {
			for {
				select {
				default:
				case <-ctx.Done():
					fmt.Println("app done.")
					return
				}

				mu.Lock()
				count++
				e.Broadcast(sse.Event{Event: "timecount", Data: strconv.FormatInt(count, 10)})
				mu.Unlock()

				time.Sleep(time.Second)
			}
		}()
		g.GET("/apis/timecount", func(c *gin.Context) {
			s, exists := e.Source(c)
			if !exists {
				c.Status(http.StatusBadRequest)
				return
			}

			if c.Query("enable") == "off" {
				collection.Delete(c)
				return
			}

			if _, exists := collection.Load(c); !exists {
				collection.Store(c, s)
			}

			mu.RLock()
			defer mu.RUnlock()
			s.Send(sse.Event{Event: "timecount", Data: strconv.FormatInt(count, 10)})
			c.Status(http.StatusOK)
		})
	}

	return e
}

func main() {
	g := gin.Default()
	g.NoRoute(noCacheFirstPage(), static.Serve("/", static.LocalFile("react-emotion/build", true)), fallback(filepath.Join("react-template/build", "index.html"), true))

	ctx := context.Background()
	e := run(ctx, g)

	srv := http.Server{
		Addr:         ":8080",
		Handler:      g,
		WriteTimeout: 0,
	}

	go func() {
		for {
			err := srv.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				time.Sleep(time.Second)
				continue
			}
			break
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	e.CloseAll()
	_ = srv.Shutdown(ctx)
}
