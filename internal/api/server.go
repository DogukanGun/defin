package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/dogukangundogan/trader/internal/config"
	"github.com/dogukangundogan/trader/internal/pool"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Server holds the HTTP API and WebSocket hub.
type Server struct {
	hub      *Hub
	registry *pool.Registry
	cfg      *config.Config
	log      *slog.Logger
}

func NewServer(hub *Hub, registry *pool.Registry, cfg *config.Config, log *slog.Logger) *Server {
	return &Server{hub: hub, registry: registry, cfg: cfg, log: log}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", s.cors(s.handleStatus))
	mux.HandleFunc("/api/pools", s.cors(s.handlePools))
	mux.HandleFunc("/api/config", s.cors(s.handleConfig))
	mux.HandleFunc("/api/balance", s.cors(s.handleBalance))
	mux.HandleFunc("/ws", s.handleWS)

	s.log.Info("API server listening", "addr", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	type statusResponse struct {
		Mode      string `json:"mode"`
		PoolCount int    `json:"pool_count"`
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statusResponse{
		Mode:      s.cfg.Execution.Mode,
		PoolCount: s.registry.Len(),
	})
}

type poolResponse struct {
	Address  string `json:"address"`
	Type     string `json:"type"`
	ChainID  int64  `json:"chain_id"`
	Token0   string `json:"token0"`
	Token1   string `json:"token1"`
	FeeBps   int    `json:"fee_bps"`
	Reserve0 string `json:"reserve0,omitempty"`
	Reserve1 string `json:"reserve1,omitempty"`
	Block    uint64 `json:"block,omitempty"`
}

func (s *Server) handlePools(w http.ResponseWriter, r *http.Request) {
	all := s.registry.All()
	resp := make([]poolResponse, 0, len(all))
	for _, p := range all {
		pr := poolResponse{
			Address: p.Address().Hex(),
			Type:    string(p.Type()),
			ChainID: p.ChainID(),
			Token0:  p.Token0().Hex(),
			Token1:  p.Token1().Hex(),
			FeeBps:  p.FeeBps(),
		}
		if st := p.State(); st != nil {
			if st.Reserve0 != nil {
				pr.Reserve0 = st.Reserve0.String()
			}
			if st.Reserve1 != nil {
				pr.Reserve1 = st.Reserve1.String()
			}
			pr.Block = st.BlockNumber
		}
		resp = append(resp, pr)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request) {
	// The hub broadcasts balance events via WebSocket; this endpoint returns the
	// last known snapshot stored in the hub's balance cache.
	addr, balances := s.hub.LastBalance()
	type resp struct {
		WalletAddress string         `json:"wallet_address"`
		Balances      []TokenBalance `json:"balances"`
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp{WalletAddress: addr, Balances: balances})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Accept and acknowledge; live reload not supported yet.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.log.Error("websocket upgrade failed", "error", err)
		return
	}

	c := &client{send: make(chan []byte, 256)}
	s.hub.register(c)
	defer s.hub.unregister(c)

	// Write pump
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case msg, ok := <-c.send:
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if !ok {
					conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// Read pump — keep connection alive and handle client messages
	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
	<-done
}
