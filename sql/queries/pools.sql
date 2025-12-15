-- name: GetRegions :many
SELECT * FROM region;

-- name: AddRegion :one
INSERT INTO region(name)
VALUES($1)
RETURNING *;

-- name: DeleteRegion :exec
DELETE FROM region as r
where r.name = $1;

-- name: GetCountries :many
SELECT * FROM country;

-- name: AddCountry :one
INSERT INTO country(name,code,region_id)
VALUES($1,$2,$3)
RETURNING *;

-- name: DeleteCountry :exec
DELETE FROM country as c
where c.name = $1;

-- name: GetUpstreams :many
SELECT * FROM upstream;

-- name: AddUpstream :one
INSERT INTO upstream(tag,upstream_provider,format,port,domain)
VALUES($1,$2,$3,$4,$5)
RETURNING *;

-- name: DeleteUpstream :exec
DELETE FROM upstream as u
where u.id = $1;

-- name: InsetPool :one
INSERT INTO pool(tag,region_id,subdomain,port)
VALUES($1,$2,$3,$4)
RETURNING *;

-- name: InsertPoolUpstreamWeight :many
INSERT INTO pool_upstream_weight (pool_id, weight, upstream_id)
SELECT $1,T.w,U.id FROM upstream AS U JOIN ROWS FROM (UNNEST($2::INT[]), UNNEST($3::text[])) AS T(w, t) ON U.tag = T.t 
RETURNING *;


-- name: ListPoolsWithUpstreams :many
SELECT 
    p.id AS pool_id,
    p.tag AS pool_tag,
    p.subdomain AS pool_subdomain,
    p.port AS pool_port,
    u.tag AS upstream_tag,
    u.format AS upstream_format,
    u.port AS upstream_port,
    u.domain AS upstream_domain
FROM pool p
LEFT JOIN pool_upstream_weight puw ON p.id = puw.pool_id
LEFT JOIN upstream u ON puw.upstream_id = u.id;

-- name: GetPoolByTagWithUpstreams :many
SELECT 
    p.id AS pool_id,
    p.tag AS pool_tag,
    p.subdomain AS pool_subdomain,
    p.port AS pool_port,
    u.tag AS upstream_tag,
    u.format AS upstream_format,
    u.port AS upstream_port,
    u.domain AS upstream_domain
FROM pool p
LEFT JOIN pool_upstream_weight puw ON p.id = puw.pool_id
LEFT JOIN upstream u ON puw.upstream_id = u.id
WHERE p.tag = $1;

-- name: UpdatePool :one
UPDATE pool
SET 
    region_id = COALESCE(sqlc.narg('region_id'), region_id),
    subdomain = COALESCE(sqlc.narg('subdomain'), subdomain),
    port = COALESCE(sqlc.narg('port'), port),
    updated_at = NOW()
WHERE tag = $1
RETURNING *;

-- name: DeletePool :exec
DELETE FROM pool
WHERE tag = $1;

-- name: AddPoolUpstreamWeight :one
INSERT INTO pool_upstream_weight (pool_id, upstream_id, weight)
VALUES (
    (SELECT p.id FROM pool p WHERE p.tag = $1),
    (SELECT u.id FROM upstream u WHERE u.tag = $2),
    $3
)
RETURNING *;

-- name: DeletePoolUpstreamWeight :exec
DELETE FROM pool_upstream_weight
WHERE pool_id = (SELECT p.id FROM pool p WHERE p.tag = $1)
  AND upstream_id = (SELECT u.id FROM upstream u WHERE u.tag = $2);
