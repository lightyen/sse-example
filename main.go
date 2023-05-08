package main

import (
	"app/sse"
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	sseSrv := sse.New()
	g.GET("/apis/stream", sseSrv.StreamHandlerFunc)

	p := sse.NewCountPlugin(ctx, sseSrv)
	sseSrv.Register(p.OnConnected)
	{
		g.GET("/apis/timecount", p.GetTimeCount(ctx, sseSrv))
	}
	return sseSrv
}

func main() {
	g := gin.Default()
	g.NoRoute(noCacheFirstPage(), static.Serve("/", static.LocalFile("react-emotion/dist", true)), fallback(filepath.Join("react-template/dist", "index.html"), true))

	ctx := context.Background()
	e := run(ctx, g)

	srv := http.Server{
		Addr:    ":8080",
		Handler: g,
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

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	e.CloseAll()
	_ = srv.Shutdown(ctx)
}
