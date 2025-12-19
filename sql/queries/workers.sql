-- name: CreateWorker :one
INSERT INTO worker (name, region_id, ip_address, port, pool_id)
VALUES ($1,(SELECT id from region where region.name = $2), $3, $4, $5)
RETURNING *;

-- name: GetAllWorkers :many
SELECT 
    w.id, 
    w.name, 
    w.ip_address, 
    w.status, 
    w.last_seen, 
    w.created_at, 
    w.port,
    w.pool_id,
    r.name AS region_name,
    COALESCE(array_agg(wd.domain) FILTER (WHERE wd.domain IS NOT NULL), '{}')::text[] AS domains
FROM worker w
JOIN region r ON w.region_id = r.id
LEFT JOIN worker_domains wd ON w.id = wd.worker_id
GROUP BY w.id, r.name;

-- name: GetWorkerByName :one
SELECT 
    w.id, 
    w.name, 
    w.ip_address, 
    w.status, 
    w.last_seen, 
    w.created_at, 
    w.port,
    w.pool_id,
    r.name AS region_name,
    COALESCE(array_agg(wd.domain) FILTER (WHERE wd.domain IS NOT NULL), '{}')::text[] AS domains
FROM worker w
JOIN region r ON w.region_id = r.id
LEFT JOIN worker_domains wd ON w.id = wd.worker_id
WHERE w.name = $1
GROUP BY w.id, r.name;

-- name: DeleteWorkerByName :exec
DELETE FROM worker WHERE name = $1;

-- name: AddWorkerDomain :one
INSERT INTO worker_domains (worker_id, domain)
VALUES ((SELECT id FROM worker WHERE name = $1), UNNEST($2::TEXT[]))
RETURNING *;

-- name: DeleteWorkerDomain :exec
DELETE FROM worker_domains
WHERE worker_id = (SELECT id FROM worker WHERE name = $1) AND domain = ANY($2::TEXT[]);

-- name: GetWorkerById :one
SELECT w.id FROM worker w
WHERE w.id = $1;



-- name: GetWorkerPoolConfig :many
SELECT 
    w.name AS worker_name,
    p.id AS pool_id,
    p.tag AS pool_tag,
    p.port AS pool_port,
    p.subdomain AS pool_subdomain,
    u.id AS upstream_id,
    u.tag AS upstream_tag,
    u.domain AS upstream_address,
    u.port AS upstream_port,
    puw.weight
FROM worker w
JOIN pool p ON w.pool_id = p.id
JOIN pool_upstream_weight puw ON p.id = puw.pool_id
JOIN upstream u ON puw.upstream_id = u.id
WHERE w.id = $1;
