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

func main() {
	e := gin.Default()
	e.NoRoute(noCacheFirstPage(), static.Serve("/", static.LocalFile("react-template/build", true)), fallback(filepath.Join("react-template/build", "index.html"), true))

	ctx := context.Background()
	evt := sse.NewEventService(ctx, e.Group("/stream"),
		sse.SingleCommand(),
		sse.TimeCount(),
	)

	srv := http.Server{
		Addr:    ":8080",
		Handler: e,
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

	evt.CloseAll()
	_ = srv.Shutdown(ctx)
}
