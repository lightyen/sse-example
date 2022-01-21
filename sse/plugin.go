package sse

import (
	"context"

	"github.com/gin-gonic/gin"
)

type Plugin interface {
	Name() string
	Setup(s *EventService, e *gin.RouterGroup) (func(p *Peer) PeerRunner, func(s *Source) SourceRunner)
	Start(c context.Context)
}

type PeerRunner interface {
	Start(c context.Context, p *Peer)
	Stop(p *Peer)
}

type SourceRunner interface {
	Start(c context.Context, s *Source)
	Stop(s *Source)
}
