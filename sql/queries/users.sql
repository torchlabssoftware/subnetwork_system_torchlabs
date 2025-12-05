-- name: CreateUser :one
INSERT INTO "user"(email,username,password,data_limit)
VALUES ($1,$2,$3,$4)
RETURNING *;

-- name: InsertUserPool :many
INSERT INTO user_pools(user_id,pool_id)
SELECT $1, UNNEST($2::uuid[])
RETURNING *;

-- name: InsertUserIpwhitelist :many
INSERT INTO user_ip_whitelist(user_id,ip_cidr)
SELECT $1, UNNEST($2::text[])
RETURNING *;

-- name: GetUserbyId :one
SELECT 
    u.id,
    u.username,
    u.password,
    u.data_usage,
    u.email,
    u.status,
    u.data_limit,
    u.created_at,
    u.updated_at,
    COALESCE(ARRAY_AGG(DISTINCT iw.ip_cidr) FILTER (WHERE iw.ip_cidr IS NOT NULL), '{}')::text[] AS ip_whitelist,
    COALESCE(ARRAY_AGG(DISTINCT p.name) FILTER (WHERE p.name IS NOT NULL), '{}')::text[] AS pools
FROM "user" AS u
LEFT JOIN user_ip_whitelist AS iw ON u.id = iw.user_id
LEFT JOIN user_pools AS up ON u.id = up.user_id
LEFT JOIN pool AS p ON up.pool_id = p.id
WHERE u.id = $1
GROUP BY u.id;

-- name: GetAllusers :many
SELECT 
    u.id,
    u.username,
    u.password,
    u.data_usage,
    u.email,
    u.status,
    u.data_limit,
    u.created_at,
    u.updated_at,
    COALESCE(ARRAY_AGG(DISTINCT iw.ip_cidr) FILTER (WHERE iw.ip_cidr IS NOT NULL), '{}')::text[] AS ip_whitelist,
    COALESCE(ARRAY_AGG(DISTINCT p.name) FILTER (WHERE p.name IS NOT NULL), '{}')::text[] AS pools
FROM "user" AS u
LEFT JOIN user_ip_whitelist AS iw ON u.id = iw.user_id
LEFT JOIN user_pools AS up ON u.id = up.user_id
LEFT JOIN pool AS p ON up.pool_id = p.id
GROUP BY u.id;

-- name: UpdateUser :one
UPDATE "user" 
SET 
email = COALESCE(sqlc.narg('email'),email), 
data_limit = COALESCE(sqlc.narg('data_limit'),data_limit), 
status = COALESCE(sqlc.narg('status'),status),
updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: SoftDeleteUser :exec
UPDATE "user" 
SET 
status = 'deleted',
updated_at = CURRENT_TIMESTAMP
WHERE id = $1;
