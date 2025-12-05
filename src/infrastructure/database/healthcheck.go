package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

// HealthStatus represents the health status of the database
type HealthStatus string

const (
	// HealthStatusHealthy indicates database is fully operational
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusDegraded indicates database is operational but with issues
	HealthStatusDegraded HealthStatus = "degraded"
	// HealthStatusUnhealthy indicates database is not operational
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheckResult contains health check information
type HealthCheckResult struct {
	Status          HealthStatus      `json:"status"`
	Latency         time.Duration     `json:"latency"`
	DatabaseVersion string            `json:"db_version,omitempty"`
	PoolStats       *ConnectionPoolStats `json:"pool_stats,omitempty"`
	Error           string            `json:"error,omitempty"`
	Timestamp       time.Time         `json:"timestamp"`
}

// ConnectionPoolStats contains connection pool statistics
type ConnectionPoolStats struct {
	ActiveConnections int `json:"active_connections"`
	MaxConnections    int `json:"max_connections"`
	IdleConnections   int `json:"idle_connections"`
	PoolUsage         float64 `json:"pool_usage_percent"`
}

// HealthChecker performs database health checks
type HealthChecker struct {
	client           *Client
	logger           *zap.Logger
	latencyThreshold time.Duration
	poolThreshold    float64
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(client *Client, logger *zap.Logger) *HealthChecker {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &HealthChecker{
		client:           client,
		logger:           logger,
		latencyThreshold: 100 * time.Millisecond,
		poolThreshold:    0.8, // 80% pool usage triggers degraded status
	}
}

// SetLatencyThreshold sets the latency threshold for health checks
func (hc *HealthChecker) SetLatencyThreshold(threshold time.Duration) {
	hc.latencyThreshold = threshold
}

// SetPoolThreshold sets the pool usage threshold (0.0 to 1.0)
func (hc *HealthChecker) SetPoolThreshold(threshold float64) {
	hc.poolThreshold = threshold
}

// Check performs a comprehensive health check
func (hc *HealthChecker) Check(ctx context.Context) *HealthCheckResult {
	result := &HealthCheckResult{
		Timestamp: time.Now(),
	}

	// Ping the database and measure latency
	latency, err := hc.Ping(ctx)
	result.Latency = latency

	if err != nil {
		result.Status = HealthStatusUnhealthy
		result.Error = err.Error()
		hc.logger.Error("Health check failed", zap.Error(err))
		return result
	}

	// Get database version
	version, err := hc.GetDatabaseVersion(ctx)
	if err == nil {
		result.DatabaseVersion = version
	}

	// Get pool stats
	poolStats, err := hc.GetPoolStats(ctx)
	if err == nil {
		result.PoolStats = poolStats
	}

	// Determine overall status
	result.Status = hc.determineStatus(latency, poolStats)

	return result
}

// Ping performs a simple ping to check database connectivity
func (hc *HealthChecker) Ping(ctx context.Context) (time.Duration, error) {
	client, err := hc.client.GetMongoClient()
	if err != nil {
		return 0, fmt.Errorf("failed to get mongo client: %w", err)
	}

	start := time.Now()
	err = client.Ping(ctx, readpref.Primary())
	latency := time.Since(start)

	if err != nil {
		return 0, fmt.Errorf("ping failed: %w", err)
	}

	return latency, nil
}

// GetDatabaseVersion retrieves the MongoDB server version
func (hc *HealthChecker) GetDatabaseVersion(ctx context.Context) (string, error) {
	db, err := hc.client.GetDefaultDatabase()
	if err != nil {
		return "", fmt.Errorf("failed to get database: %w", err)
	}

	var result bson.M
	err = db.RunCommand(ctx, bson.D{{Key: "buildInfo", Value: 1}}).Decode(&result)
	if err != nil {
		return "", fmt.Errorf("failed to get build info: %w", err)
	}

	version, ok := result["version"].(string)
	if !ok {
		return "", fmt.Errorf("version not found in build info")
	}

	return version, nil
}

// GetPoolStats retrieves connection pool statistics
func (hc *HealthChecker) GetPoolStats(ctx context.Context) (*ConnectionPoolStats, error) {
	db, err := hc.client.GetDefaultDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	var result bson.M
	err = db.RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}}).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to get server status: %w", err)
	}

	connections, ok := result["connections"].(bson.M)
	if !ok {
		return &ConnectionPoolStats{}, nil
	}

	stats := &ConnectionPoolStats{}

	if current, ok := connections["current"].(int32); ok {
		stats.ActiveConnections = int(current)
	}

	if available, ok := connections["available"].(int32); ok {
		stats.IdleConnections = int(available)
	}

	// Calculate max connections (active + idle)
	stats.MaxConnections = stats.ActiveConnections + stats.IdleConnections

	// Calculate pool usage percentage
	if stats.MaxConnections > 0 {
		stats.PoolUsage = float64(stats.ActiveConnections) / float64(stats.MaxConnections)
	}

	return stats, nil
}

// determineStatus determines the overall health status
func (hc *HealthChecker) determineStatus(latency time.Duration, poolStats *ConnectionPoolStats) HealthStatus {
	// Check if latency exceeds threshold
	if latency > hc.latencyThreshold {
		hc.logger.Warn("High database latency detected",
			zap.Duration("latency", latency),
			zap.Duration("threshold", hc.latencyThreshold),
		)
		return HealthStatusDegraded
	}

	// Check if pool usage exceeds threshold
	if poolStats != nil && poolStats.PoolUsage > hc.poolThreshold {
		hc.logger.Warn("High connection pool usage detected",
			zap.Float64("usage", poolStats.PoolUsage),
			zap.Float64("threshold", hc.poolThreshold),
		)
		return HealthStatusDegraded
	}

	return HealthStatusHealthy
}

// CheckReplicaSet checks replica set health (if applicable)
func (hc *HealthChecker) CheckReplicaSet(ctx context.Context) (bool, int, error) {
	db, err := hc.client.GetDefaultDatabase()
	if err != nil {
		return false, 0, fmt.Errorf("failed to get database: %w", err)
	}

	var result bson.M
	err = db.RunCommand(ctx, bson.D{{Key: "isMaster", Value: 1}}).Decode(&result)
	if err != nil {
		return false, 0, fmt.Errorf("failed to check replica set: %w", err)
	}

	// Check if this is a replica set
	isReplicaSet := false
	if setName, ok := result["setName"].(string); ok && setName != "" {
		isReplicaSet = true
	}

	// Count hosts if replica set
	nodeCount := 0
	if isReplicaSet {
		if hosts, ok := result["hosts"].(bson.A); ok {
			nodeCount = len(hosts)
		}
	}

	return isReplicaSet, nodeCount, nil
}

// StartContinuousMonitoring starts continuous health monitoring
func (hc *HealthChecker) StartContinuousMonitoring(ctx context.Context, interval time.Duration, resultChan chan<- *HealthCheckResult) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	hc.logger.Info("Starting continuous health monitoring",
		zap.Duration("interval", interval),
	)

	for {
		select {
		case <-ctx.Done():
			hc.logger.Info("Stopping continuous health monitoring")
			return
		case <-ticker.C:
			result := hc.Check(ctx)
			select {
			case resultChan <- result:
			default:
				hc.logger.Warn("Health check result channel full, dropping result")
			}
		}
	}
}

// GetMetrics returns a map of all health metrics
func (hc *HealthChecker) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	result := hc.Check(ctx)

	metrics := map[string]interface{}{
		"db_connection_status": string(result.Status),
		"db_latency_ms":        result.Latency.Milliseconds(),
		"db_version":           result.DatabaseVersion,
	}

	if result.PoolStats != nil {
		metrics["pool_active_connections"] = result.PoolStats.ActiveConnections
		metrics["pool_max_connections"] = result.PoolStats.MaxConnections
		metrics["pool_idle_connections"] = result.PoolStats.IdleConnections
		metrics["pool_usage_percent"] = result.PoolStats.PoolUsage * 100
	}

	if result.Error != "" {
		metrics["error"] = result.Error
	}

	return metrics, nil
}

// ShouldAlert determines if an alert should be triggered based on thresholds
func (hc *HealthChecker) ShouldAlert(result *HealthCheckResult) bool {
	if result.Status == HealthStatusUnhealthy {
		return true
	}

	if result.Latency > hc.latencyThreshold {
		return true
	}

	if result.PoolStats != nil && result.PoolStats.PoolUsage > hc.poolThreshold {
		return true
	}

	return false
}
