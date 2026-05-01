package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	sdkmath "cosmossdk.io/math"
	"github.com/gorilla/mux"

	"github.com/cognize/axon/poaa-engine/consensus"
	"github.com/cognize/axon/poaa-engine/dag"
)

type Server struct {
	router  *mux.Router
	engine  *consensus.Engine
	memDAG  interface{}
	server  *http.Server
	mu      sync.RWMutex
	started bool
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string     `json:"error,omitempty"`
}

type VertexResponse struct {
	Hash        string   `json:"hash"`
	Parents     []string `json:"parents"`
	Timestamp   int64    `json:"timestamp"`
	Index       uint64   `json:"index"`
	Sender      string   `json:"sender"`
	Layer       uint8    `json:"layer"`
	Confirmed   bool     `json:"confirmed"`
	TotalWeight string   `json:"total_weight"`
	Depth       uint64   `json:"depth"`
}

type StatusResponse struct {
	Height      uint64 `json:"height"`
	Round       uint64 `json:"round"`
	IsRunning   bool   `json:"is_running"`
	TipsCount   int    `json:"tips_count"`
	Validators  int    `json:"validators"`
}

func NewServer(engine *consensus.Engine, memDAG interface{}) *Server {
	s := &Server{
		engine: engine,
		memDAG: memDAG,
		router: mux.NewRouter(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.HandleFunc("/status", s.handleStatus).Methods("GET")
	s.router.HandleFunc("/vertex/{hash}", s.handleGetVertex).Methods("GET")
	s.router.HandleFunc("/tips", s.handleGetTips).Methods("GET")
	s.router.HandleFunc("/validators", s.handleGetValidators).Methods("GET")
	s.router.HandleFunc("/height", s.handleGetHeight).Methods("GET")
	s.router.HandleFunc("/vertex", s.handleSubmitVertex).Methods("POST")
	s.router.HandleFunc("/confirm/{hash}", s.handleConfirm).Methods("POST")
	s.router.HandleFunc("/finalize/{hash}", s.handleFinalize).Methods("POST")
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	height, round, size := s.engine.GetState()

	resp := StatusResponse{
		Height:     height,
		Round:      round,
		IsRunning:  s.engine.IsRunning(),
		TipsCount:  size,
		Validators: len(s.engine.GetValidators()),
	}

	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: resp})
}

func (s *Server) handleGetVertex(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hashStr := vars["hash"]

	var hash [32]byte
	if _, err := fmt.Sscanf(hashStr, "%x", &hash); err != nil {
		s.writeError(w, "invalid hash format", http.StatusBadRequest)
		return
	}

	v, err := s.engine.GetVertex(hash)
	if err != nil {
		s.writeError(w, fmt.Sprintf("vertex not found: %v", err), http.StatusNotFound)
		return
	}

	resp := vertexToResponse(v)
	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: resp})
}

func (s *Server) handleGetTips(w http.ResponseWriter, r *http.Request) {
	tips := s.engine.GetTips()

	resp := make([]VertexResponse, 0, len(tips))
	for _, t := range tips {
		resp = append(resp, vertexToResponse(t))
	}

	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: resp})
}

func (s *Server) handleGetValidators(w http.ResponseWriter, r *http.Request) {
	validators := s.engine.GetValidators()

	type ValidatorResp struct {
		Address    string `json:"address"`
		Power      string `json:"power"`
		Reputation int64  `json:"reputation"`
	}

	resp := make([]ValidatorResp, 0, len(validators))
	for _, v := range validators {
		resp = append(resp, ValidatorResp{
			Address:    v.Address.String(),
			Power:      v.Power.String(),
			Reputation: v.Reputation,
		})
	}

	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: resp})
}

func (s *Server) handleGetHeight(w http.ResponseWriter, r *http.Request) {
	height := s.engine.GetHeight()

	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: map[string]uint64{"height": height}})
}

func (s *Server) handleSubmitVertex(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TxBytes   string   `json:"tx_bytes"`
		Parents   []string `json:"parents"`
		Sender    string   `json:"sender"`
		Timestamp int64    `json:"timestamp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	v := &dag.Vertex{
		Sender:    req.Sender,
		Timestamp: req.Timestamp,
		Layer:     dag.LayerSoft,
	}

	if len(req.TxBytes) > 0 {
		v.TxBytes = []byte(req.TxBytes)
	}

	if len(req.Parents) > 0 {
		for i, p := range req.Parents {
			if i >= 2 {
				break
			}
			var hash [32]byte
			fmt.Sscanf(p, "%x", &hash)
			v.Parents[i] = hash
		}
	}

	v.Hash = v.ComputeHash()

	if err := s.engine.SubmitVertex(v); err != nil {
		s.writeError(w, fmt.Sprintf("failed to submit vertex: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSON(w, http.StatusOK, Response{Success: true, Data: vertexToResponse(v)})
}

func (s *Server) handleConfirm(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hashStr := vars["hash"]

	var hash [32]byte
	if _, err := fmt.Sscanf(hashStr, "%x", &hash); err != nil {
		s.writeError(w, "invalid hash format", http.StatusBadRequest)
		return
	}

	var req struct {
		Agent      string `json:"agent"`
		Reputation int64  `json:"reputation"`
		Weight     string `json:"weight"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	weight, _ := sdkmath.LegacyNewDecFromStr(req.Weight)
	if err := s.engine.ConfirmVertex(hash, req.Agent, req.Reputation, weight); err != nil {
		s.writeError(w, fmt.Sprintf("failed to confirm: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSON(w, http.StatusOK, Response{Success: true})
}

func (s *Server) handleFinalize(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hashStr := vars["hash"]

	var hash [32]byte
	if _, err := fmt.Sscanf(hashStr, "%x", &hash); err != nil {
		s.writeError(w, "invalid hash format", http.StatusBadRequest)
		return
	}

	if err := s.engine.FinalizeVertex(hash); err != nil {
		s.writeError(w, fmt.Sprintf("failed to finalize: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSON(w, http.StatusOK, Response{Success: true})
}

func vertexToResponse(v *dag.Vertex) VertexResponse {
	return VertexResponse{
		Hash:        fmt.Sprintf("%x", v.Hash),
		Parents:     []string{fmt.Sprintf("%x", v.Parents[0]), fmt.Sprintf("%x", v.Parents[1])},
		Timestamp:   v.Timestamp,
		Index:       v.Index,
		Sender:      v.Sender,
		Layer:       uint8(v.Layer),
		Confirmed:   v.Confirmed,
		TotalWeight: v.TotalWeight.String(),
		Depth:       v.Depth,
	}
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, msg string, status int) {
	s.writeJSON(w, status, Response{Success: false, Error: msg})
}

func (s *Server) Start(addr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("server already started")
	}

	s.server = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	s.started = true
	go func() {
		s.server.ListenAndServe()
	}()

	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	s.started = false
	return s.server.Close()
}

func (s *Server) Router() *mux.Router {
	return s.router
}
