-- name: CreateWorker :one
INSERT INTO worker (name, region_id, ip_address)
VALUES ($1,(SELECT id from region where region.name = $2), $3)
RETURNING *;

