package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
)

type AnalyticsService interface {
	RecordUserDataUsage(ctx context.Context, data UserDataUsage) error
	RecordWorkerHealth(ctx context.Context, data WorkerHealth) error
	RecordWebsiteAccess(ctx context.Context, data WebsiteAccess) error
	GetUserUsage(ctx context.Context, userID string, from, to time.Time, granularity string) (interface{}, error)
	GetWorkerHealth(ctx context.Context, workerID string, from, to time.Time) ([]WorkerHealth, error)
	GetUserWebsiteAccess(ctx context.Context, userID string, from, to time.Time) ([]WebsiteAccess, error)
	StartWorkers()
}

type analyticsService struct {
	conn              driver.Conn
	userDataChan      chan UserDataUsage
	websiteAccessChan chan WebsiteAccess
}

func NewAnalyticsService(conn driver.Conn) AnalyticsService {
	return &analyticsService{
		conn:              conn,
		userDataChan:      make(chan UserDataUsage, 10000),
		websiteAccessChan: make(chan WebsiteAccess, 10000),
	}
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
	SourceIP        string    `json:"source_ip"`
	Protocol        string    `json:"protocol"`
	DestinationHost string    `json:"destination_host"`
	DestinationPort uint16    `json:"destination_port"`
	StatusCode      uint16    `json:"status_code"`
}

type UpstreamHealth struct {
	UpstreamID  uuid.UUID `json:"upstream_id"`
	UpstreamTag string    `json:"upstream_tag"`
	Status      string    `json:"status"`
	Latency     int64     `json:"latency"`
	ErrorRate   float32   `json:"error_rate"`
}

type WorkerHealth struct {
	WorkerID              uuid.UUID        `json:"worker_id"`
	WorkerName            string           `json:"worker_name"`
	Region                string           `json:"region"`
	Status                string           `json:"status"`
	CpuUsage              float32          `json:"cpu_usage"`
	MemoryUsage           float32          `json:"memory_usage"`
	ActiveConnections     uint32           `json:"active_connections"`
	TotalConnections      uint64           `json:"total_connections"`
	BytesThroughputPerSec uint64           `json:"bytes_throughput_per_sec"`
	ErrorRate             float32          `json:"error_rate"`
	Upstreams             []UpstreamHealth `json:"upstreams"`
}

func (s *analyticsService) RecordUserDataUsage(ctx context.Context, data UserDataUsage) error {
	// Validate SourceIP
	if data.SourceIP == "" {
		data.SourceIP = "0.0.0.0"
	}

	select {
	case s.userDataChan <- data:
		return nil
	default:
		// Drop event if buffer is full
		return fmt.Errorf("analytics buffer full, dropping user data event")
	}
}

func (s *analyticsService) RecordWorkerHealth(ctx context.Context, data WorkerHealth) error {
	// 1. Record Worker App Health
	queryWorker := `
		INSERT INTO analytics.worker_health (
			worker_id, worker_name, region, status, cpu_usage, memory_usage,
			active_connections, total_connections, bytes_throughput_per_sec, error_rate
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
	`
	if err := s.conn.Exec(ctx, queryWorker,
		data.WorkerID, data.WorkerName, data.Region, data.Status, data.CpuUsage, data.MemoryUsage,
		data.ActiveConnections, data.TotalConnections, data.BytesThroughputPerSec, data.ErrorRate,
	); err != nil {
		return fmt.Errorf("failed to insert worker health: %w", err)
	}

	// 2. Record Upstream Health
	if len(data.Upstreams) > 0 {
		batch, err := s.conn.PrepareBatch(ctx, "INSERT INTO analytics.worker_upstream_health (worker_id, upstream_id, upstream_tag, status, latency, error_rate)")
		if err != nil {
			return fmt.Errorf("failed to prepare batch for upstreams: %w", err)
		}

		for _, u := range data.Upstreams {
			if err := batch.Append(
				data.WorkerID,
				u.UpstreamID,
				u.UpstreamTag,
				u.Status,
				u.Latency,
				u.ErrorRate,
			); err != nil {
				return fmt.Errorf("failed to append upstream health to batch: %w", err)
			}
		}

		if err := batch.Send(); err != nil {
			return fmt.Errorf("failed to send upstream health batch: %w", err)
		}
	}

	return nil
}

type WebsiteAccess struct {
	UserID        uuid.UUID `json:"user_id"`
	Username      string    `json:"username"`
	Domain        string    `json:"domain"`
	Subdomain     string    `json:"subdomain"`
	FullURL       string    `json:"full_url"`
	BytesSent     uint64    `json:"bytes_sent"`
	BytesReceived uint64    `json:"bytes_received"`
	RequestMethod string    `json:"request_method"`
	StatusCode    uint16    `json:"status_code"`
	ContentType   string    `json:"content_type"`

	SourceIP string `json:"source_ip"`
}

func (s *analyticsService) RecordWebsiteAccess(ctx context.Context, data WebsiteAccess) error {
	// Validate SourceIP
	if data.SourceIP == "" {
		data.SourceIP = "0.0.0.0"
	}

	select {
	case s.websiteAccessChan <- data:
		return nil
	default:
		return fmt.Errorf("analytics buffer full, dropping website access event")
	}
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

func (s *analyticsService) GetWorkerHealth(ctx context.Context, workerID string, from, to time.Time) ([]WorkerHealth, error) {
	query := `
		SELECT 
			worker_id, worker_name, region, status, cpu_usage, memory_usage,
			active_connections, total_connections, bytes_throughput_per_sec, error_rate
		FROM analytics.worker_health
		WHERE worker_id = ? AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp
		LIMIT 1000
	`
	var results []WorkerHealth
	if err := s.conn.Select(ctx, &results, query, workerID, from, to); err != nil {
		return nil, err
	}
	return results, nil
}

func (s *analyticsService) GetUserWebsiteAccess(ctx context.Context, userID string, from, to time.Time) ([]WebsiteAccess, error) {
	query := `
		SELECT
			user_id, username, domain, subdomain, full_url,
			bytes_sent, bytes_received, request_method, status_code,
			content_type, source_ip
		FROM analytics.website_access
		WHERE user_id = ? AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp
		LIMIT 1000
	`
	var results []WebsiteAccess
	if err := s.conn.Select(ctx, &results, query, userID, from, to); err != nil {
		return nil, err
	}
	return results, nil
}

func (s *analyticsService) StartWorkers() {
	go s.processUserDataBatch()
	go s.processWebsiteAccessBatch()
}

func (s *analyticsService) processUserDataBatch() {
	batchSize := 1000
	flushInterval := 5 * time.Second

	batch := make([]UserDataUsage, 0, batchSize)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case item := <-s.userDataChan:
			batch = append(batch, item)
			if len(batch) >= batchSize {
				s.flushUserData(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				s.flushUserData(batch)
				batch = batch[:0]
			}
		}
	}
}

func (s *analyticsService) flushUserData(items []UserDataUsage) {
	if len(items) == 0 {
		return
	}

	ctx := context.Background()
	query := `INSERT INTO analytics.user_data_usage (
			user_id, username, pool_id, pool_name, worker_id, worker_region,
			bytes_sent, bytes_received, source_ip,
			protocol, destination_host, destination_port, status_code
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)`

	batch, err := s.conn.PrepareBatch(ctx, query)
	if err != nil {
		log.Printf("Failed to prepare user data batch: %v", err)
		return
	}

	for _, data := range items {
		err := batch.Append(
			data.UserID, data.Username, data.PoolID, data.PoolName, data.WorkerID, data.WorkerRegion,
			data.BytesSent, data.BytesReceived, data.SourceIP,
			data.Protocol, data.DestinationHost, data.DestinationPort, data.StatusCode,
		)
		if err != nil {
			log.Printf("Failed to append user data to batch: %v", err)
			continue
		}
	}

	if err := batch.Send(); err != nil {
		log.Printf("Failed to send user data batch: %v", err)
	}
}

func (s *analyticsService) processWebsiteAccessBatch() {
	batchSize := 1000
	flushInterval := 5 * time.Second

	batch := make([]WebsiteAccess, 0, batchSize)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case item := <-s.websiteAccessChan:
			batch = append(batch, item)
			if len(batch) >= batchSize {
				s.flushWebsiteAccess(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				s.flushWebsiteAccess(batch)
				batch = batch[:0]
			}
		}
	}
}

func (s *analyticsService) flushWebsiteAccess(items []WebsiteAccess) {
	if len(items) == 0 {
		return
	}

	ctx := context.Background()
	query := `INSERT INTO analytics.website_access (
			user_id, username, domain, subdomain, full_url,
			bytes_sent, bytes_received, request_method, status_code,
			content_type, source_ip
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)`

	batch, err := s.conn.PrepareBatch(ctx, query)
	if err != nil {
		log.Printf("Failed to prepare website access batch: %v", err)
		return
	}

	for _, data := range items {
		err := batch.Append(
			data.UserID, data.Username, data.Domain, data.Subdomain, data.FullURL,
			data.BytesSent, data.BytesReceived, data.RequestMethod, data.StatusCode,
			data.ContentType, data.SourceIP,
		)
		if err != nil {
			log.Printf("Failed to append website access to batch: %v", err)
			continue
		}
	}

	if err := batch.Send(); err != nil {
		log.Printf("Failed to send website access batch: %v", err)
	}
}
