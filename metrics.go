/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Device metrics
	devicesConnected = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "rttys_devices_connected",
		Help: "Number of currently connected devices",
	}, []string{"group"})

	devicesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rttys_devices_connections_total",
		Help: "Total number of device connections since server start",
	})

	deviceDisconnects = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rttys_devices_disconnections_total",
		Help: "Total number of device disconnections since server start",
	})

	// User/session metrics
	activeUserSessions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "rttys_user_sessions_active",
		Help: "Number of active user WebSocket sessions",
	})

	userSessionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rttys_user_sessions_total",
		Help: "Total number of user sessions since server start",
	})

	// HTTP proxy metrics
	httpProxySessionsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "rttys_http_proxy_sessions_active",
		Help: "Number of active HTTP proxy sessions",
	})

	httpProxyRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rttys_http_proxy_requests_total",
		Help: "Total number of HTTP proxy requests",
	})

	// Command execution metrics
	commandExecutionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rttys_command_executions_total",
		Help: "Total number of command executions",
	}, []string{"status"})

	commandExecutionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "rttys_command_execution_duration_seconds",
		Help:    "Duration of command executions in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// WebSocket metrics
	websocketMessagesReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rttys_websocket_messages_received_total",
		Help: "Total WebSocket messages received from users",
	})

	websocketMessagesSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "rttys_websocket_messages_sent_total",
		Help: "Total WebSocket messages sent to users",
	})

	// File transfer metrics
	fileTransfersTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "rttys_file_transfers_total",
		Help: "Total number of file transfers",
	}, []string{"direction"}) // "upload" or "download"
)

// MetricsIncDeviceConnected increments device connected metrics
func MetricsIncDeviceConnected(group string) {
	devicesConnected.WithLabelValues(group).Inc()
	devicesTotal.Inc()
}

// MetricsDecDeviceConnected decrements device connected gauge
func MetricsDecDeviceConnected(group string) {
	devicesConnected.WithLabelValues(group).Dec()
	deviceDisconnects.Inc()
}

// MetricsIncUserSession increments user session metrics
func MetricsIncUserSession() {
	activeUserSessions.Inc()
	userSessionsTotal.Inc()
}

// MetricsDecUserSession decrements user session gauge
func MetricsDecUserSession() {
	activeUserSessions.Dec()
}

// MetricsIncHttpProxySession increments HTTP proxy session count
func MetricsIncHttpProxySession() {
	httpProxySessionsActive.Inc()
	httpProxyRequestsTotal.Inc()
}

// MetricsDecHttpProxySession decrements HTTP proxy session count
func MetricsDecHttpProxySession() {
	httpProxySessionsActive.Dec()
}

// MetricsRecordCommandExecution records a command execution
func MetricsRecordCommandExecution(status string, duration float64) {
	commandExecutionsTotal.WithLabelValues(status).Inc()
	commandExecutionDuration.Observe(duration)
}

// MetricsIncWebSocketReceived increments WebSocket messages received
func MetricsIncWebSocketReceived() {
	websocketMessagesReceived.Inc()
}

// MetricsIncWebSocketSent increments WebSocket messages sent
func MetricsIncWebSocketSent() {
	websocketMessagesSent.Inc()
}

// MetricsIncFileTransfer increments file transfer counter
func MetricsIncFileTransfer(direction string) {
	fileTransfersTotal.WithLabelValues(direction).Inc()
}

