package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	models "github.com/torchlabssoftware/subnetwork_system/internal/server/models"
)

type analyticsService struct {
	conn              driver.Conn
	userDataChan      chan models.UserDataUsage
	websiteAccessChan chan models.WebsiteAccess
}

func NewAnalyticsService(conn driver.Conn) models.AnalyticsService {
	return &analyticsService{
		conn:              conn,
		userDataChan:      make(chan models.UserDataUsage, 10000),
		websiteAccessChan: make(chan models.WebsiteAccess, 10000),
	}
}

func (s *analyticsService) RecordUserDataUsage(ctx context.Context, data models.UserDataUsage) error {
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

func (s *analyticsService) RecordWorkerHealth(ctx context.Context, data models.WorkerHealth) error {
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

func (s *analyticsService) RecordWebsiteAccess(ctx context.Context, data models.WebsiteAccess) error {
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
		var results []models.UserUsageHourly
		if err := s.conn.Select(ctx, &results, query, userID, from, to); err != nil {
			return nil, err
		}
		return results, nil
	}
	return nil, fmt.Errorf("unsupported granularity: %s", granularity)
}

func (s *analyticsService) GetWorkerHealth(ctx context.Context, workerID string, from, to time.Time) ([]models.WorkerHealth, error) {
	query := `
		SELECT 
			worker_id, worker_name, region, status, cpu_usage, memory_usage,
			active_connections, total_connections, bytes_throughput_per_sec, error_rate
		FROM analytics.worker_health
		WHERE worker_id = ? AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp
		LIMIT 1000
	`
	var results []models.WorkerHealth
	if err := s.conn.Select(ctx, &results, query, workerID, from, to); err != nil {
		return nil, err
	}
	return results, nil
}

func (s *analyticsService) GetUserWebsiteAccess(ctx context.Context, userID string, from, to time.Time) ([]models.WebsiteAccess, error) {
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
	var results []models.WebsiteAccess
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

	batch := make([]models.UserDataUsage, 0, batchSize)
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

func (s *analyticsService) flushUserData(items []models.UserDataUsage) {
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

	batch := make([]models.WebsiteAccess, 0, batchSize)
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

func (s *analyticsService) flushWebsiteAccess(items []models.WebsiteAccess) {
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
