-- name: GetPoolsbyTags :many
SELECT * FROM pool
WHERE pool.tag = ANY($1::text[]);