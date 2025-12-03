-- name: CreateUser :one
INSERT INTO "user"(email,username,password,data_limit)
VALUES ($1,$2,$3,$4)
RETURNING *;

-- name: InsertUserPool :one
INSERT INTO user_pools(pool_id,user_id)
values ($1,$2)
RETURNING *;

-- name: InsertUserIpwhitelist :one
INSERT INTO user_ip_whitelist(user_id,ip_cidr)
VALUES($1,$2)
RETURNING *;