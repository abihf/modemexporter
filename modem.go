package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Modem interface {
	Auth(ctx context.Context, user, password string) (*time.Time, error)
	Stat(ctx context.Context) (*Stat, error)
}

type Stat struct {
	TxBytes   uint64
	TxPackets uint64
	RxBytes   uint64
	RxPackets uint64
}

type Server struct {
	Modem      Modem
	User       string
	Password   string
	authExpire time.Time
	mu         sync.Mutex
}

var _ http.Handler = &Server{}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/plain")

	ctx := r.Context()
	err := s.auth(ctx)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	stat, err := s.Modem.Stat(ctx)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	statMap := map[string]uint64{
		"rx_bytes":   stat.RxBytes,
		"rx_packets": stat.RxPackets,
		"tx_bytes":   stat.TxBytes,
		"tx_packets": stat.TxPackets,
	}
	for name, value := range statMap {
		fmt.Fprintf(w, "# TYPE modem_%s counter\n", name)
		fmt.Fprintf(w, "modem_%s %d\n", name, value)
	}
}

func (s *Server) auth(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if time.Now().After(s.authExpire) {
		expire, err := s.Modem.Auth(ctx, s.User, s.Password)
		if err != nil {
			return err
		}
		s.authExpire = *expire
	}

	return nil
}
