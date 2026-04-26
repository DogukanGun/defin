package telemetry

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	BlocksProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trader_blocks_processed_total",
			Help: "Total number of blocks processed",
		},
		[]string{"chain"},
	)

	OpportunitiesFound = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trader_opportunities_found_total",
			Help: "Total opportunities found by strategy",
		},
		[]string{"chain", "strategy"},
	)

	OpportunitiesExecuted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trader_opportunities_executed_total",
			Help: "Total opportunities executed",
		},
		[]string{"chain", "strategy", "status"},
	)

	BlockProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trader_block_processing_duration_seconds",
			Help:    "Time to process a block",
			Buckets: []float64{0.01, 0.025, 0.05, 0.075, 0.1, 0.15, 0.2, 0.25, 0.5},
		},
		[]string{"chain"},
	)

	MulticallDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trader_multicall_duration_seconds",
			Help:    "Time for multicall batch read",
			Buckets: []float64{0.01, 0.025, 0.05, 0.075, 0.1},
		},
		[]string{"chain"},
	)

	ProfitTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trader_profit_wei_total",
			Help: "Total profit in wei",
		},
		[]string{"chain", "strategy"},
	)
)

func init() {
	prometheus.MustRegister(
		BlocksProcessed,
		OpportunitiesFound,
		OpportunitiesExecuted,
		BlockProcessingDuration,
		MulticallDuration,
		ProfitTotal,
	)
}

func ServeMetrics(port int) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}
