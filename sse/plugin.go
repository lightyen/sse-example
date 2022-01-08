package sse

import (
	"context"

	"github.com/gin-gonic/gin"
)

type Plugin interface {
	Name() string
	Setup(s *EventService, e *gin.RouterGroup) (func(p *Peer) PeerRunner, func(s *Source) SourceRunner)
	Serve(c context.Context)
}

type PeerRunner interface {
	Run(c context.Context, p *Peer)
	Stop(p *Peer)
}

type SourceRunner interface {
	Run(c context.Context, s *Source)
	Stop(s *Source)
}
