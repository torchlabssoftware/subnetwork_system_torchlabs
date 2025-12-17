-- Create analytics database
CREATE DATABASE IF NOT EXISTS analytics;

-- User data usage (detailed granularity)
CREATE TABLE IF NOT EXISTS analytics.user_data_usage (
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
    session_id String,
    source_ip IPv4,
    user_agent String,
    protocol String,
    destination_host String,
    destination_port UInt16,
    status_code UInt16
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (user_id, date, timestamp)
TTL date + INTERVAL 90 DAY;

-- Hourly aggregated user usage (for fast queries)
CREATE TABLE IF NOT EXISTS analytics.user_usage_hourly (
    date Date,
    hour DateTime,
    user_id UUID,
    username String,
    bytes_sent AggregateFunction(sum, UInt64),
    bytes_received AggregateFunction(sum, UInt64),
    request_count AggregateFunction(count, UInt64),
    unique_sessions AggregateFunction(uniq, String),
    unique_destinations AggregateFunction(uniq, String)
) ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (user_id, date, hour);

-- Daily aggregated user usage
CREATE TABLE IF NOT EXISTS analytics.user_usage_daily (
    date Date,
    user_id UUID,
    username String,
    bytes_sent UInt64,
    bytes_received UInt64,
    request_count UInt64,
    unique_sessions UInt64,
    unique_destinations UInt64
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (user_id, date);

-- Pool performance metrics
CREATE TABLE IF NOT EXISTS analytics.pool_performance (
    timestamp DateTime64(3) DEFAULT now64(3),
    date Date DEFAULT toDate(timestamp),
    pool_id UUID,
    pool_name String,
    upstream_domain String,
    region String,
    request_count UInt64,
    success_count UInt64,
    failed_count UInt64,
    avg_response_time_ms Float32,
    p95_response_time_ms Float32,
    p99_response_time_ms Float32,
    total_bytes UInt64,
    unique_users_count UInt64
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (pool_id, date, timestamp)
TTL date + INTERVAL 180 DAY;

-- Website access patterns
CREATE TABLE IF NOT EXISTS analytics.website_access (
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
    user_agent String,
    session_id String,
    source_ip IPv4
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (domain, date, timestamp)
TTL date + INTERVAL 30 DAY;

-- Worker health metrics
CREATE TABLE IF NOT EXISTS analytics.worker_health (
    timestamp DateTime64(3) DEFAULT now64(3),
    date Date DEFAULT toDate(timestamp),
    worker_id UUID,
    worker_name String,
    region String,
    status String,
    cpu_usage Float32,
    memory_usage Float32,
    active_connections UInt32,
    total_connections UInt64,
    bytes_throughput_per_sec UInt64,
    upstream_health Map(String, Float32), -- Map of upstream domain to health score
    error_rate Float32,
    last_heartbeat DateTime
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (worker_id, date, timestamp)
TTL date + INTERVAL 60 DAY;

-- User behavior analytics
CREATE TABLE IF NOT EXISTS analytics.user_behavior (
    timestamp DateTime64(3) DEFAULT now64(3),
    date Date DEFAULT toDate(timestamp),
    user_id UUID,
    username String,
    session_id String,
    session_start DateTime,
    session_end DateTime,
    total_requests UInt32,
    total_bytes UInt64,
    avg_requests_per_minute Float32,
    peak_bandwidth_mbps Float32,
    common_protocols Array(String),
    top_domains Array(Tuple(String, UInt32)), -- (domain, count)
    geolocation String,
    is_bot Boolean DEFAULT 0,
    user_category String -- casual, power, business, etc.
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (user_id, date, timestamp)
TTL date + INTERVAL 365 DAY;

-- Create materialized view for hourly aggregation
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.user_usage_hourly_mv
TO analytics.user_usage_hourly
AS
SELECT
    toDate(timestamp) AS date,
    toStartOfHour(timestamp) AS hour,
    user_id,
    username,
    sumState(bytes_sent) AS bytes_sent,
    sumState(bytes_received) AS bytes_received,
    countState() AS request_count,
    uniqState(session_id) AS unique_sessions,
    uniqState(destination_host) AS unique_destinations
FROM analytics.user_data_usage
GROUP BY date, hour, user_id, username;

-- Create view for daily aggregation
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.user_usage_daily_mv
TO analytics.user_usage_daily
AS
SELECT
    date,
    user_id,
    username,
    sum(bytes_sent) AS bytes_sent,
    sum(bytes_received) AS bytes_received,
    count() AS request_count,
    uniq(session_id) AS unique_sessions,
    uniq(destination_host) AS unique_destinations
FROM analytics.user_data_usage
GROUP BY date, user_id, username;

-- Create user for analytics
CREATE USER IF NOT EXISTS analytics IDENTIFIED WITH sha256_password BY 'analytics123';
GRANT ALL ON analytics.* TO analytics;
