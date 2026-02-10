-- Create analytics database
CREATE DATABASE IF NOT EXISTS analytics_db_subnetworksystem;

-- User data usage (detailed granularity)
CREATE TABLE IF NOT EXISTS analytics_db_subnetworksystem.user_data_usage (
    timestamp DateTime64(3) DEFAULT now64(3),
    date Date DEFAULT toDate(timestamp),
    user_id UUID,
    username String,
    pool_id UUID,
    pool_name String,
    worker_id UUID,
    worker_region String,
    bytes_sent UInt64,
    bytes_received UInt64,
    source_ip String,
    protocol String,
    destination_host String,
    destination_port UInt16,
    status_code UInt16
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (user_id, date, timestamp)
TTL date + INTERVAL 90 DAY;

-- Hourly aggregated user usage (for fast queries)
CREATE TABLE IF NOT EXISTS analytics_db_subnetworksystem.user_usage_hourly (
    date Date,
    hour DateTime,
    user_id UUID,
    username String,
    bytes_sent AggregateFunction(sum, UInt64),
    bytes_received AggregateFunction(sum, UInt64),
    request_count AggregateFunction(count, UInt64),
    unique_destinations AggregateFunction(uniq, String)
) ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (user_id, date, hour);

-- Daily aggregated user usage
CREATE TABLE IF NOT EXISTS analytics_db_subnetworksystem.user_usage_daily (
    date Date,
    user_id UUID,
    username String,
    bytes_sent AggregateFunction(sum, UInt64),
    bytes_received AggregateFunction(sum, UInt64),
    request_count AggregateFunction(count, UInt64),
    unique_destinations AggregateFunction(uniq, String)
) ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (user_id, date);

-- Worker health metrics
CREATE TABLE IF NOT EXISTS analytics_db_subnetworksystem.worker_health (
    timestamp DateTime64(3) DEFAULT now64(3),
    date Date DEFAULT toDate(timestamp),
    worker_id UUID,
    worker_name String,
    region String,
    status String,
    pool_tag String,
    cpu_usage Float32,
    memory_usage Float32,
    active_connections UInt32,
    total_connections UInt64,
    bytes_throughput_per_sec UInt64,
    upstream_health Map(String, Float32),
    error_rate Float32
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (worker_id, date, timestamp)
TTL date + INTERVAL 60 DAY;

-- Upstream Health metrics
CREATE TABLE IF NOT EXISTS analytics_db_subnetworksystem.worker_upstream_health (
    timestamp DateTime64(3) DEFAULT now64(3),
    date Date DEFAULT toDate(timestamp),
    worker_id UUID,
    upstream_id UUID,
    upstream_tag String,
    status String,
    latency Int64,
    error_rate Float32
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (worker_id, upstream_id, date, timestamp)
TTL date + INTERVAL 60 DAY;

-- Website access patterns
CREATE TABLE IF NOT EXISTS analytics_db_subnetworksystem.website_access (
    timestamp DateTime64(3) DEFAULT now64(3),
    date Date DEFAULT toDate(timestamp),
    user_id UUID,
    username String,
    domain String,
    subdomain String,
    full_url String,
    bytes_sent UInt64,
    bytes_received UInt64,
    request_method String,
    status_code UInt16,
    content_type String,
    source_ip String
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (domain, date, timestamp)
TTL date + INTERVAL 30 DAY;

-- Create materialized view for hourly aggregation
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics_db_subnetworksystem.user_usage_hourly_mv
TO analytics_db_subnetworksystem.user_usage_hourly
AS
SELECT
    toDate(timestamp) AS date,
    toStartOfHour(timestamp) AS hour,
    user_id,
    username,
    sumState(bytes_sent) AS bytes_sent,
    sumState(bytes_received) AS bytes_received,
    countState() AS request_count,
    uniqState(destination_host) AS unique_destinations
FROM analytics_db_subnetworksystem.user_data_usage
GROUP BY date, hour, user_id, username;

-- Create view for daily aggregation
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics_db_subnetworksystem.user_usage_daily_mv
TO analytics_db_subnetworksystem.user_usage_daily
AS
SELECT
    date,
    user_id,
    username,
    sumState(bytes_sent) AS bytes_sent,
    sumState(bytes_received) AS bytes_received,
    countState() AS request_count,
    uniqState(destination_host) AS unique_destinations
FROM analytics_db_subnetworksystem.user_data_usage
GROUP BY date, user_id, username;

