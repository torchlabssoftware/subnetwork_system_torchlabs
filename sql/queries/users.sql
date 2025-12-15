-- name: CreateUser :one
INSERT INTO "user"(username,password)
VALUES ($1,$2)
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
    u.status,
    u.created_at,
    u.updated_at,
    COALESCE(ARRAY_AGG(DISTINCT iw.ip_cidr) FILTER (WHERE iw.ip_cidr IS NOT NULL), '{}')::text[] AS ip_whitelist,
    COALESCE(ARRAY_AGG(DISTINCT p.tag) FILTER (WHERE p.tag IS NOT NULL), '{}')::text[] AS pools
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
    u.status,
    u.created_at,
    u.updated_at,
    COALESCE(ARRAY_AGG(DISTINCT iw.ip_cidr) FILTER (WHERE iw.ip_cidr IS NOT NULL), '{}')::text[] AS ip_whitelist,
    COALESCE(ARRAY_AGG(DISTINCT p.tag) FILTER (WHERE p.tag IS NOT NULL), '{}')::text[] AS pools
FROM "user" AS u
LEFT JOIN user_ip_whitelist AS iw ON u.id = iw.user_id
LEFT JOIN user_pools AS up ON u.id = up.user_id
LEFT JOIN pool AS p ON up.pool_id = p.id
GROUP BY u.id;

-- name: UpdateUser :one
UPDATE "user" 
SET 
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

-- name: GetDatausageById :many
SELECT up.data_limit,up.data_usage,p.tag AS pool_tag 
FROM user_pools AS up 
INNER JOIN pool AS p ON up.pool_id = p.id
WHERE up.user_id = $1;

-- name: GetUserPoolsByUserId :one
select u.id,
    COALESCE(ARRAY_AGG(DISTINCT up.pool_id) FILTER (WHERE up.pool_id IS NOT NULL),'{}')::TEXT[] as pool_ids,
    COALESCE(ARRAY_AGG(up.data_limit) FILTER (WHERE up.data_limit IS NOT NULL),'{}')::BIGINT[] AS data_limits,
    COALESCE(ARRAY_AGG(up.data_usage) FILTER (WHERE up.data_usage IS NOT NULL),'{}')::BIGINT[]  AS data_usages
from  "user" as u
join user_pools as up
on u.id = up.user_id
WHERE u.id = $1
group by u.id;

-- name: AddUserPoolsByPoolTags :one
WITH matching_pools AS (
    SELECT id, tag
    FROM pool 
    WHERE tag = ANY($2::text[])
), 
inserted_rows AS (
    INSERT INTO user_pools (user_id, pool_id,data_limit)
    SELECT $1, id,UNNEST($3::BIGINT[]) FROM matching_pools
    ON CONFLICT (user_id, pool_id) DO NOTHING
    RETURNING pool_id, user_id,data_limit
)
SELECT 
    i.user_id, 
    ARRAY_AGG(p.tag)::TEXT[] AS inserted_tags,
    ARRAY_AGG(i.data_limit)::BIGINT[] AS inserted_data_limits
FROM inserted_rows i
JOIN matching_pools p ON i.pool_id = p.id
GROUP BY i.user_id;

-- name: DeleteUserPoolsByTags :exec
DELETE FROM user_pools
WHERE user_id = $1
  AND pool_id IN (
      SELECT id 
      FROM pool 
      WHERE tag = ANY($2::text[]) 
  );

-- name: GetUserIpwhitelistByUserId :one
SELECT 
    u.id AS user_id, 
    COALESCE(
        ARRAY_AGG(DISTINCT w.ip_cidr) FILTER (WHERE w.ip_cidr IS NOT NULL), 
        '{}'
    )::TEXT[] AS ip_list
FROM "user" u                       
LEFT JOIN user_ip_whitelist w      
    ON u.id = w.user_id
WHERE u.id = $1
GROUP BY u.id;

-- name: DeleteUserIpwhitelist :exec
DELETE FROM user_ip_whitelist
WHERE user_id = $1
  AND ip_cidr = ANY($2::TEXT[]);

-- name: GetUserByUsername :one
SELECT *
FROM "user"
WHERE username = $1;

-- name: GenerateproxyString :one
SELECT p.tag,p.subdomain,p.port,u.username,u.password FROM pool as p
join region as r on p.region_id = r.id
join country as c on r.id = c.region_id
join user_pools as up on p.id = up.pool_id
join "user" as u on up.user_id = u.id
where c.code = $1 AND p.tag LIKE $2 AND up.user_id = $3 ;






