-- name: CreateUser :one
INSERT INTO "user"(id,email,username,password,data_limit,data_usage,status,created_at,updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
RETURNING *;