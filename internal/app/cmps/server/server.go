package server

import (
	"context"
	"time"

	"github.com/martketplace-vkr/pkg/server/http"
)

const (
	cmpName = "Http server cmp"
)

type Server struct {
	cfg Config

	server *http.FiberServer
}

func New(cfg Config, server *http.FiberServer) *Server {
	return &Server{
		cfg:    cfg,
		server: server,
	}
}

func (s *Server) Start(ctx context.Context) (err error) {
	err = s.server.Start(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) (err error) {
	err = s.server.Stop(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) GetStartTimeout() time.Duration {
	return s.cfg.StartTimeout.Duration
}

func (s *Server) GetStopTimeout() time.Duration {
	return s.cfg.StopTimeout.Duration
}

func (s *Server) GetShutdownDelay() time.Duration {
	return s.cfg.ShutdownDelay.Duration
}

func (s *Server) GetName() string {
	return cmpName
}
