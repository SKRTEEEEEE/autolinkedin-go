package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// HealthChecker defines interface for database health check
type HealthChecker interface {
	Check(ctx context.Context) *HealthCheckResult
}

// HealthCheckResult represents health check result
type HealthCheckResult struct {
	Status  string        `json:"status"`
	Latency time.Duration `json:"latency"`
	Error   string        `json:"error,omitempty"`
}

// WorkerRegistry defines interface for worker status tracking
type WorkerRegistry interface {
	GetStatus() map[string]*WorkerStatus
	IsHealthy() bool
}

// WorkerStatus represents worker status
type WorkerStatus struct {
	Name      string
	Running   bool
	StartedAt time.Time
	Error     error
}

// NATSClient defines interface for NATS connection status
type NATSClient interface {
	IsConnected() bool
}

// HealthHandler handles health check requests
type HealthHandler struct {
	dbHealthChecker HealthChecker
	workerRegistry  WorkerRegistry
	natsClient      NATSClient
	logger          *zap.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(
	dbHealthChecker HealthChecker,
	workerRegistry WorkerRegistry,
	natsClient NATSClient,
	logger *zap.Logger,
) *HealthHandler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &HealthHandler{
		dbHealthChecker: dbHealthChecker,
		workerRegistry:  workerRegistry,
		natsClient:      natsClient,
		logger:          logger,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status     string                   `json:"status"`
	Timestamp  time.Time                `json:"timestamp"`
	Components HealthComponentsResponse `json:"components"`
}

// HealthComponentsResponse represents the status of each component
type HealthComponentsResponse struct {
	Database string                      `json:"database"`
	NATS     string                      `json:"nats"`
	Workers  map[string]WorkerHealthInfo `json:"workers"`
}

// WorkerHealthInfo represents worker health information
type WorkerHealthInfo struct {
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// HandleHealth handles GET /health requests
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check database health
	dbStatus := "unknown"
	if h.dbHealthChecker != nil {
		dbHealth := h.dbHealthChecker.Check(ctx)
		dbStatus = dbHealth.Status
	}

	// Check NATS connection
	natsStatus := "disconnected"
	if h.natsClient != nil && h.natsClient.IsConnected() {
		natsStatus = "connected"
	}

	// Check worker status
	workerHealth := make(map[string]WorkerHealthInfo)
	overallWorkerStatus := "healthy"

	if h.workerRegistry != nil {
		workerStatuses := h.workerRegistry.GetStatus()
		for name, ws := range workerStatuses {
			status := "stopped"
			errorMsg := ""

			if ws.Running {
				status = "running"
			} else if ws.Error != nil {
				status = "failed"
				errorMsg = ws.Error.Error()
				overallWorkerStatus = "unhealthy"
			} else {
				overallWorkerStatus = "degraded"
			}

			workerHealth[name] = WorkerHealthInfo{
				Status:    status,
				StartedAt: ws.StartedAt,
				Error:     errorMsg,
			}
		}
	}

	// Determine overall status
	overallStatus := "healthy"
	statusCode := http.StatusOK

	if dbStatus == "unhealthy" || natsStatus == "disconnected" || overallWorkerStatus == "unhealthy" {
		overallStatus = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	} else if dbStatus == "degraded" || overallWorkerStatus == "degraded" {
		overallStatus = "degraded"
		statusCode = http.StatusOK // Still OK, but degraded
	}

	// Build response
	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Components: HealthComponentsResponse{
			Database: dbStatus,
			NATS:     natsStatus,
			Workers:  workerHealth,
		},
	}

	// Log health check result
	h.logger.Debug("Health check completed",
		zap.String("status", overallStatus),
		zap.String("db_status", dbStatus),
		zap.String("nats_status", natsStatus),
		zap.String("worker_status", overallWorkerStatus),
	)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode health response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// HandleReadiness handles GET /ready requests (for k8s readiness probes)
func (h *HealthHandler) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if all critical components are ready
	ready := true

	// Check database
	if h.dbHealthChecker != nil {
		dbHealth := h.dbHealthChecker.Check(ctx)
		if dbHealth.Status == "unhealthy" {
			ready = false
		}
	}

	// Check NATS
	if h.natsClient != nil && !h.natsClient.IsConnected() {
		ready = false
	}

	// Check workers
	if h.workerRegistry != nil && !h.workerRegistry.IsHealthy() {
		ready = false
	}

	if ready {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("not ready"))
	}
}

// HandleLiveness handles GET /live requests (for k8s liveness probes)
func (h *HealthHandler) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	// Liveness probe just checks if the application is running
	// It should not check external dependencies
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("alive"))
}
