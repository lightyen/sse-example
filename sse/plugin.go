package sse

import (
	"context"

	"github.com/gin-gonic/gin"
)

type Plugin interface {
	Install(s *EventService, e *gin.RouterGroup) func(s *EventSource) PluginInstance
	// Run plugin task
	Run(c context.Context)
}

type PluginInstance interface {
	// Run with the certain client
	Run(s *EventSource)
	// Dispose the certain client
	Dispose(s *EventSource)
}
