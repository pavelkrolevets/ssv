package topics

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

var (
	metricPubsubTrace = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ssv:network:pubsub:trace",
		Help: "Traces of pubsub messages",
	}, []string{"type"})
	metricPubsubMsgValidationResults = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ssv:network:pubsub:msg:validation",
		Help: "Traces of pubsub message validation results",
	}, []string{"type"})
	metricPubsubOutbound = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ssv:p2p:pubsub:msg:out",
		Help: "Count broadcasted messages",
	}, []string{"topic"})
	metricPubsubInbound = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ssv:p2p:pubsub:msg:in",
		Help: "Count incoming messages",
	}, []string{"topic", "msg_type"})
	metricPubsubActiveMsgValidation = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ssv:p2p:pubsub:msg:val:active",
		Help: "Count active message validation",
	}, []string{"topic"})
	metricPubsubPeerScoreInspect = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ssv:p2p:pubsub:score:inspect",
		Help: "Gauge for negative peer scores",
	}, []string{"pid"})
	duplicatesRemovedCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "number_of_duplicates_removed",
			Help: "Count the number of times a duplicate signature set has been removed.",
		},
	)
	numberOfSetsAggregated = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "number_of_sets_aggregated",
			Help:    "Count the number of times different sets have been successfully aggregated in a batch.",
			Buckets: []float64{10, 50, 100, 200, 400, 800, 1600, 3200},
		},
	)
)

func init() {
	if err := prometheus.Register(metricPubsubTrace); err != nil {
		log.Println("could not register prometheus collector")
	}
	if err := prometheus.Register(metricPubsubMsgValidationResults); err != nil {
		log.Println("could not register prometheus collector")
	}
	if err := prometheus.Register(metricPubsubOutbound); err != nil {
		log.Println("could not register prometheus collector")
	}
	if err := prometheus.Register(metricPubsubInbound); err != nil {
		log.Println("could not register prometheus collector")
	}
	if err := prometheus.Register(metricPubsubActiveMsgValidation); err != nil {
		log.Println("could not register prometheus collector")
	}
	if err := prometheus.Register(metricPubsubPeerScoreInspect); err != nil {
		log.Println("could not register prometheus collector")
	}
}

type msgValidationResult string

var (
	validationResultNoData         msgValidationResult = "no_data"
	validationResultEncoding       msgValidationResult = "encoding"
	validationResultNoValidator    msgValidationResult = "no_validator"
	ValidationResultNotTimely      msgValidationResult = "not_timely"
	ValidationResultSyntacticCheck msgValidationResult = "syntactic_check"
	ValidationResultBetterMessage  msgValidationResult = "better_message"
	ValidationResultInvalidSig     msgValidationResult = "invalid signature"
	ValidationResultTooManyMsgs    msgValidationResult = "too_many_msgs"
	validationResultOk             msgValidationResult = "valid"
)

func reportValidationResult(result msgValidationResult, logger *zap.Logger, err error, msg string) {
	if err != nil {
		logger.Error(string(result), zap.String("msg", msg), zap.Error(err))
	} else {
		logger.Info(string(result), zap.String("msg", msg))
	}
	metricPubsubMsgValidationResults.WithLabelValues(string(result)).Inc()
}
