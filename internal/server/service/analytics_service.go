package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
)

type AnalyticsService interface {
	RecordUserDataUsage(ctx context.Context, data UserDataUsage) error
	RecordWorkerHealth(ctx context.Context, data WorkerHealth) error
	GetUserUsage(ctx context.Context, userID string, from, to time.Time, granularity string) (interface{}, error)
}

type analyticsService struct {
	conn driver.Conn
}

func NewAnalyticsService(conn driver.Conn) AnalyticsService {
	return &analyticsService{conn: conn}
}

type UserDataUsage struct {
	UserID          uuid.UUID `json:"user_id"`
	Username        string    `json:"username"`
	PoolID          uuid.UUID `json:"pool_id"`
	PoolName        string    `json:"pool_name"`
	WorkerID        uuid.UUID `json:"worker_id"`
	WorkerRegion    string    `json:"worker_region"`
	BytesSent       uint64    `json:"bytes_sent"`
	BytesReceived   uint64    `json:"bytes_received"`
	SessionID       string    `json:"session_id"`
	SourceIP        string    `json:"source_ip"`
	UserAgent       string    `json:"user_agent"`
	Protocol        string    `json:"protocol"`
	DestinationHost string    `json:"destination_host"`
	DestinationPort uint16    `json:"destination_port"`
	StatusCode      uint16    `json:"status_code"`
}

type WorkerHealth struct {
	WorkerID              uuid.UUID `json:"worker_id"`
	WorkerName            string    `json:"worker_name"`
	Region                string    `json:"region"`
	Status                string    `json:"status"`
	CpuUsage              float32   `json:"cpu_usage"`
	MemoryUsage           float32   `json:"memory_usage"`
	ActiveConnections     uint32    `json:"active_connections"`
	TotalConnections      uint64    `json:"total_connections"`
	BytesThroughputPerSec uint64    `json:"bytes_throughput_per_sec"`
	ErrorRate             float32   `json:"error_rate"`
}

func (s *analyticsService) RecordUserDataUsage(ctx context.Context, data UserDataUsage) error {
	// Validate SourceIP
	if data.SourceIP == "" {
		data.SourceIP = "0.0.0.0"
	}

	query := `
		INSERT INTO analytics.user_data_usage (
			user_id, username, pool_id, pool_name, worker_id, worker_region,
			bytes_sent, bytes_received, session_id, source_ip, user_agent,
			protocol, destination_host, destination_port, status_code
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
	`
	return s.conn.Exec(ctx, query,
		data.UserID, data.Username, data.PoolID, data.PoolName, data.WorkerID, data.WorkerRegion,
		data.BytesSent, data.BytesReceived, data.SessionID, data.SourceIP, data.UserAgent,
		data.Protocol, data.DestinationHost, data.DestinationPort, data.StatusCode,
	)
}

func (s *analyticsService) RecordWorkerHealth(ctx context.Context, data WorkerHealth) error {
	query := `
		INSERT INTO analytics.worker_health (
			worker_id, worker_name, region, status, cpu_usage, memory_usage,
			active_connections, total_connections, bytes_throughput_per_sec, error_rate
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
	`
	return s.conn.Exec(ctx, query,
		data.WorkerID, data.WorkerName, data.Region, data.Status, data.CpuUsage, data.MemoryUsage,
		data.ActiveConnections, data.TotalConnections, data.BytesThroughputPerSec, data.ErrorRate,
	)
}

type UserUsageHourly struct {
	Date               time.Time `ch:"date" json:"date"`
	Hour               time.Time `ch:"hour" json:"hour"`
	UserID             uuid.UUID `ch:"user_id" json:"user_id"`
	Username           string    `ch:"username" json:"username"`
	BytesSent          uint64    `ch:"bytes_sent" json:"bytes_sent"`
	BytesReceived      uint64    `ch:"bytes_received" json:"bytes_received"`
	RequestCount       uint64    `ch:"request_count" json:"request_count"`
	UniqueSessions     uint64    `ch:"unique_sessions" json:"unique_sessions"`
	UniqueDestinations uint64    `ch:"unique_destinations" json:"unique_destinations"`
}

func (s *analyticsService) GetUserUsage(ctx context.Context, userID string, from, to time.Time, granularity string) (interface{}, error) {
	if granularity == "hour" {
		query := `
			SELECT 
				date, hour, user_id, username,
				sumMerge(bytes_sent) as bytes_sent,
				sumMerge(bytes_received) as bytes_received,
				countMerge(request_count) as request_count,
				uniqMerge(unique_sessions) as unique_sessions,
				uniqMerge(unique_destinations) as unique_destinations
			FROM analytics.user_usage_hourly
			WHERE user_id = ? AND date >= ? AND date <= ?
			GROUP BY date, hour, user_id, username
			ORDER BY hour
		`
		var results []UserUsageHourly
		if err := s.conn.Select(ctx, &results, query, userID, from, to); err != nil {
			return nil, err
		}
		return results, nil
	}
	return nil, fmt.Errorf("unsupported granularity: %s", granularity)
}
