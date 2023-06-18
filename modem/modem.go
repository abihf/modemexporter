package modem

import (
	"context"
	"fmt"
	"net/http"
)

type Modem interface {
	Info(ctx context.Context) (*Info, error)
	Stat(ctx context.Context) (*Stat, error)
}

type Info struct {
	Vendor string
	Model  string

	Extra []byte
}
type Stat struct {
	TxBytes   uint64
	TxPackets uint64
	RxBytes   uint64
	RxPackets uint64

	Extra []byte
}

type Server struct {
	Modem Modem
}

var _ http.Handler = &Server{}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/plain")

	ctx := r.Context()

	if s.Modem == nil {
		http.Error(w, "Modem not set", 500)
		return
	}

	info, err := s.Modem.Info(ctx)
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
		fmt.Fprintf(w, "modem_%s{vendor=\"%s\",model=\"%s\"} %d\n", name, info.Vendor, info.Model, value)
	}
}
