-- +goose up

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE region (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE country (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    region_id UUID NOT NULL REFERENCES region(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE "user" (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE,
    username TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    data_limit BIGINT NOT NULL DEFAULT 0,
    data_usage BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleted')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_ip_whitelist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    ip_cidr TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE pool (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    tag TEXT,
    region_id UUID REFERENCES region(id) ON DELETE SET NULL,
    subdomain TEXT,
    port INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_pools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pool_id UUID NOT NULL REFERENCES pool(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    UNIQUE(pool_id, user_id)
);

CREATE TABLE upstream (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    format TEXT NOT NULL,
    port INT NOT NULL,
    domain TEXT NOT NULL,
    pool_id UUID NOT NULL REFERENCES pool(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE pool_upstream_weight (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pool_id UUID NOT NULL REFERENCES pool(id) ON DELETE CASCADE,
    upstream_id UUID NOT NULL REFERENCES upstream(id) ON DELETE CASCADE,
    weight INT NOT NULL DEFAULT 1,
    UNIQUE(pool_id, upstream_id)
);

CREATE TABLE worker (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    region_id UUID NOT NULL REFERENCES region(id) ON DELETE CASCADE,
    ip_address TEXT NOT NULL,
    status TEXT DEFAULT 'active',
    last_seen TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE worker_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_id UUID NOT NULL REFERENCES worker(id) ON DELETE CASCADE,
    domain TEXT NOT NULL,
    UNIQUE(worker_id, domain)
);

-- +goose down
DROP TABLE worker_domains;
DROP TABLE worker;
DROP TABLE pool_upstream_weight;
DROP TABLE upstream;
DROP TABLE user_pools;
DROP TABLE pool;
DROP TABLE user_ip_whitelist;
DROP TABLE "user";
DROP TABLE country;
DROP TABLE region;
