package metrics

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	once     sync.Once
	registry *prometheus.Registry

	HeightGauge *prometheus.GaugeVec
	RoundGauge  *prometheus.GaugeVec
	TxCounter   *prometheus.CounterVec
	ConfirmCounter *prometheus.CounterVec
	FinalizeCounter *prometheus.CounterVec
	LatencyHistogram *prometheus.HistogramVec

	VertexGauge *prometheus.GaugeVec
	TipsGauge   *prometheus.GaugeVec
	ValidatorGauge *prometheus.GaugeVec
	ReputationGauge *prometheus.GaugeVec

	VRFProposerGauge *prometheus.GaugeVec
	BLSAggregationGauge *prometheus.GaugeVec

	StoreOperationsCounter *prometheus.CounterVec
	StoreLatencyHistogram *prometheus.HistogramVec

	P2PConnectionsGauge *prometheus.GaugeVec
	P2PMessagesCounter *prometheus.CounterVec
)

func initMetrics() {
	registry = prometheus.NewRegistry()

	HeightGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cognize_height",
		Help: "Current block height",
	}, []string{"chain"})

	RoundGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cognize_round",
		Help: "Current consensus round",
	}, []string{"chain"})

	TxCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cognize_transactions_total",
		Help: "Total number of transactions",
	}, []string{"chain", "type"})

	ConfirmCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cognize_confirmations_total",
		Help: "Total number of confirmations",
	}, []string{"chain", "layer"})

	FinalizeCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cognize_finalizations_total",
		Help: "Total number of finalizations",
	}, []string{"chain"})

	LatencyHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "cognize_latency_seconds",
		Help:    "Latency in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"chain", "operation"})

	VertexGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cognize_vertices",
		Help: "Number of vertices in DAG",
	}, []string{"chain", "layer"})

	TipsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cognize_tips",
		Help: "Number of tips in DAG",
	}, []string{"chain"})

	ValidatorGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cognize_validators",
		Help: "Number of active validators",
	}, []string{"chain"})

	ReputationGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cognize_reputation",
		Help: "Validator reputation score",
	}, []string{"chain", "address"})

	VRFProposerGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cognize_vrf_proposers",
		Help: "VRF proposer selections",
	}, []string{"chain", "address"})

	BLSAggregationGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cognize_bls_aggregations",
		Help: "BLS signature aggregations",
	}, []string{"chain"})

	StoreOperationsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cognize_store_operations_total",
		Help: "Total store operations",
	}, []string{"chain", "operation"})

	StoreLatencyHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "cognize_store_latency_seconds",
		Help:    "Store operation latency",
		Buckets: prometheus.DefBuckets,
	}, []string{"chain", "operation"})

	P2PConnectionsGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "cognize_p2p_connections",
		Help: "Number of P2P connections",
	}, []string{"chain"})

	P2PMessagesCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "cognize_p2p_messages_total",
		Help: "Total P2P messages",
	}, []string{"chain", "type"})

	registry.MustRegister(HeightGauge)
	registry.MustRegister(RoundGauge)
	registry.MustRegister(TxCounter)
	registry.MustRegister(ConfirmCounter)
	registry.MustRegister(FinalizeCounter)
	registry.MustRegister(LatencyHistogram)
	registry.MustRegister(VertexGauge)
	registry.MustRegister(TipsGauge)
	registry.MustRegister(ValidatorGauge)
	registry.MustRegister(ReputationGauge)
	registry.MustRegister(VRFProposerGauge)
	registry.MustRegister(BLSAggregationGauge)
	registry.MustRegister(StoreOperationsCounter)
	registry.MustRegister(StoreLatencyHistogram)
	registry.MustRegister(P2PConnectionsGauge)
	registry.MustRegister(P2PMessagesCounter)
}

func GetRegistry() *prometheus.Registry {
	once.Do(initMetrics)
	return registry
}

func Handler() http.Handler {
	return promhttp.HandlerFor(GetRegistry(), promhttp.HandlerOpts{})
}

func ObserveLatency(chain, operation string, duration float64) {
	GetRegistry()
	LatencyHistogram.WithLabelValues(chain, operation).Observe(duration)
}

func IncTx(chain, txType string) {
	GetRegistry()
	TxCounter.WithLabelValues(chain, txType).Inc()
}

func IncConfirm(chain, layer string) {
	GetRegistry()
	ConfirmCounter.WithLabelValues(chain, layer).Inc()
}

func IncFinalize(chain string) {
	GetRegistry()
	FinalizeCounter.WithLabelValues(chain).Inc()
}

func SetHeight(chain string, height float64) {
	GetRegistry()
	HeightGauge.WithLabelValues(chain).Set(height)
}

func SetRound(chain string, round float64) {
	GetRegistry()
	RoundGauge.WithLabelValues(chain).Set(round)
}

func SetTips(chain string, count float64) {
	GetRegistry()
	TipsGauge.WithLabelValues(chain).Set(count)
}

func SetValidators(chain string, count float64) {
	GetRegistry()
	ValidatorGauge.WithLabelValues(chain).Set(count)
}

func SetReputation(chain, address string, value float64) {
	GetRegistry()
	ReputationGauge.WithLabelValues(chain, address).Set(value)
}

func SetVertices(chain, layer string, count float64) {
	GetRegistry()
	VertexGauge.WithLabelValues(chain, layer).Set(count)
}

func IncVRFProposer(chain, address string) {
	GetRegistry()
	VRFProposerGauge.WithLabelValues(chain, address).Inc()
}

func SetBLSAggregations(chain string, count float64) {
	GetRegistry()
	BLSAggregationGauge.WithLabelValues(chain).Set(count)
}

func IncStoreOperation(chain, operation string) {
	GetRegistry()
	StoreOperationsCounter.WithLabelValues(chain, operation).Inc()
}

func ObserveStoreLatency(chain, operation string, duration float64) {
	GetRegistry()
	StoreLatencyHistogram.WithLabelValues(chain, operation).Observe(duration)
}

func SetP2PConnections(chain string, count float64) {
	GetRegistry()
	P2PConnectionsGauge.WithLabelValues(chain).Set(count)
}

func IncP2PMessage(chain, msgType string) {
	GetRegistry()
	P2PMessagesCounter.WithLabelValues(chain, msgType).Inc()
}

type MetricsServer struct {
	addr   string
	server *http.Server
}

func NewMetricsServer(addr string) *MetricsServer {
	return &MetricsServer{
		addr: addr,
	}
}

func (m *MetricsServer) Start() error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", Handler())

	m.server = &http.Server{
		Addr:    m.addr,
		Handler: mux,
	}

	return m.server.ListenAndServe()
}

func (m *MetricsServer) Stop() error {
	if m.server != nil {
		return m.server.Close()
	}
	return nil
}

func RecordConsensusRound(chain string, height, round uint64, validatorCount int) {
	SetHeight(chain, float64(height))
	SetRound(chain, float64(round))
	SetValidators(chain, float64(validatorCount))
}

func RecordVertexAdded(chain, layer string) {
	IncTx(chain, "vertex")
	SetVertices(chain, layer, 1)
}

func RecordVertexConfirmed(chain, layer string) {
	IncConfirm(chain, layer)
}

func RecordVertexFinalized(chain string) {
	IncFinalize(chain)
}
