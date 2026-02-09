BEGIN;

----------------------------------------------------------
-- 1. Clear existing data
----------------------------------------------------------
TRUNCATE TABLE 
    worker_domains,
    worker,
    pool_upstream_weight,
    upstream,
    user_pools,
    pool,
    user_ip_whitelist,
    "user",
    country,
    region
CASCADE;

----------------------------------------------------------
-- 2. Regions
----------------------------------------------------------
INSERT INTO region (name) VALUES
('North America'),
('Europe'),
('Asia');

----------------------------------------------------------
-- 3. Countries
----------------------------------------------------------
INSERT INTO country (name, code, region_id) VALUES
('United States', 'US', (SELECT id FROM region WHERE name = 'North America')),
('Canada',        'CA', (SELECT id FROM region WHERE name = 'North America')),
('Germany',       'DE', (SELECT id FROM region WHERE name = 'Europe')),
('France',        'FR', (SELECT id FROM region WHERE name = 'Europe')),
('Japan',         'JP', (SELECT id FROM region WHERE name = 'Asia')),
('Singapore',     'SG', (SELECT id FROM region WHERE name = 'Asia'));

----------------------------------------------------------
-- 4. Users
----------------------------------------------------------
INSERT INTO "user" (username, password, status) VALUES
('gr74gtr4', '82k51ebu', 'active'),
('345rw3r5', 'alom2rkt', 'active'),
('fgn9i8wd', '1670omer', 'active');

----------------------------------------------------------
-- 5. IP Whitelist
----------------------------------------------------------
INSERT INTO user_ip_whitelist (user_id, ip_cidr)
SELECT u.id, w.ip
FROM "user" u
CROSS JOIN (
    VALUES
    ('192.168.1.0/24'),
    ('10.0.0.0/8'),
    ('203.0.113.0/24')
) AS w(ip)
WHERE u.username IN ('345rw3r5', 'gr74gtr4');

----------------------------------------------------------
-- 6. Pools
----------------------------------------------------------
INSERT INTO pool (tag, region_id, subdomain, port) VALUES
('netnutusa',        (SELECT id FROM region WHERE name='North America'), 'netnut.usa',       7000),
('netnuteu',         (SELECT id FROM region WHERE name='Europe'),        'netnut.eu',        7001),
('netnutasia',       (SELECT id FROM region WHERE name='Asia'),          'netnut.asia',      7002),

('netnutsocks5usa',  (SELECT id FROM region WHERE name='North America'), 'socksnetnut.usa',  7003),
('netnutsocks5eu',   (SELECT id FROM region WHERE name='Europe'),        'socksnetnut.eu',   7004),
('netnutsocks5asia', (SELECT id FROM region WHERE name='Asia'),          'socksnetnut.asia', 7005),

('iproyalusa',       (SELECT id FROM region WHERE name='North America'), 'iproyal.usa',      8000),
('iproyaleu',        (SELECT id FROM region WHERE name='Europe'),        'iproyal.eu',       8001),
('iproyalasia',      (SELECT id FROM region WHERE name='Asia'),          'iproyal.asia',     8002),

('geonodeusa',       (SELECT id FROM region WHERE name='North America'), '136.116.66.60',     9000),
('geonodeeu',        (SELECT id FROM region WHERE name='Europe'),        '34.88.145.246',     9001),
('geonodeasia',      (SELECT id FROM region WHERE name='Asia'),          '34.131.224.30',   9002);

----------------------------------------------------------
-- 7. Assign Users to Pools
----------------------------------------------------------
INSERT INTO user_pools (pool_id, user_id, data_limit)
SELECT p.id, u.id, 1000
FROM pool p
JOIN "user" u ON TRUE
WHERE
    u.username = 'gr74gtr4'
    OR (u.username = '345rw3r5' AND p.tag LIKE 'netnut%')
    OR (u.username = 'fgn9i8wd' AND p.tag LIKE 'geonode%');

----------------------------------------------------------
-- 8. Upstreams
----------------------------------------------------------
INSERT INTO upstream (tag, upstream_provider,username,password, config_format, port, domain) VALUES
('netnutusa', 'netnut', 'cFAPhxyG', '9dgbjKKV', '-res-[country]-sid-[session]', 6500, 'netnut.x.proxiess.com'),
('netnutsocks5usa', 'netnut', 'cFAPhxyG', '9dgbjKKV', '-res-[country]-sid-[session]', 6503, 'socksnetnut.x.proxiess.com'),
('geonodeusa', 'geonode', 'pxy_ud5azgr1', '06e4ddd1-e608-4be6-971b-bef71b53ac1e', '-country-[country]-session-[session]-lifetime-60', 9000, 'premium-residential.geonode.com'),
('iproyalusa', 'iproyal','otJhMuv0', '5uhhT0Ds', '-country-[country]_session-[session]_lifetime-1h', 9789, 'boilingresi.x.proxiess.com'),

('netnuteu', 'netnut','uscFAPhxyGer', '9dgbjKKV', '-res-[country]-sid-[session]', 6501, 'netnutasia.x.proxiess.com'),
('geonodeeu', 'geonode', 'pxy_ud5azgr1', '06e4ddd1-e608-4be6-971b-bef71b53ac1e', '-country-[country]-session-[session]-lifetime-60', 9000, 'premium-residential.geonode.com'),
('iproyaleu', 'iproyal','otJhMuv0', '5uhhT0Ds', '-country-[country]_session-[session]_lifetime-1h', 9790, 'boilingresieu.x.proxiess.com'),

('netnutasia', 'netnut','cFAPhxyG', '9dgbjKKV', '-res-[country]-sid-[session]', 6502, 'netnuteu.x.proxiess.com'),
('geonodeasia', 'geonode', 'pxy_ud5azgr1', '06e4ddd1-e608-4be6-971b-bef71b53ac1e', '-country-[country]-session-[session]-lifetime-60', 9000, 'premium-residential.geonode.com'),
('iproyalasia', 'iproyal','otJhMuv0', '5uhhT0Ds', '-country-[country]_session-[session]_lifetime-1h', 9791, 'boilingresiasia.x.proxiess.com');


----------------------------------------------------------
-- 9. Upstream Weights (FIXED)
----------------------------------------------------------
INSERT INTO pool_upstream_weight (pool_id, upstream_id, weight)
VALUES
((SELECT id FROM pool WHERE tag='netnutusa'), (SELECT id FROM upstream WHERE tag='netnutusa'), 50),
((SELECT id FROM pool WHERE tag='netnutsocks5usa'), (SELECT id FROM upstream WHERE tag='netnutsocks5usa'), 50),
((SELECT id FROM pool WHERE tag='netnuteu'), (SELECT id FROM upstream WHERE tag='netnuteu'), 50),
((SELECT id FROM pool WHERE tag='netnutasia'), (SELECT id FROM upstream WHERE tag='netnutasia'), 50),
((SELECT id FROM pool WHERE tag='geonodeusa'), (SELECT id FROM upstream WHERE tag='geonodeusa'), 50),
((SELECT id FROM pool WHERE tag='geonodeeu'), (SELECT id FROM upstream WHERE tag='geonodeeu'), 50),
((SELECT id FROM pool WHERE tag='geonodeasia'), (SELECT id FROM upstream WHERE tag='geonodeasia'), 50);

----------------------------------------------------------
-- 10. Workers
----------------------------------------------------------
INSERT INTO worker (name, region_id, ip_address,status, pool_id,port, last_seen) VALUES
('usa-00000000000000000000000000000000', (SELECT id FROM region WHERE name='North America'), '34.67.96.169', 'active', (SELECT id FROM pool WHERE tag='geonodeusa'),9000, NOW()),
('usa-11111111111111111111111111111111', (SELECT id FROM region WHERE name='North America'), '34.67.96.169', 'active', (SELECT id FROM pool WHERE tag='netnutsocks5usa'),7003, NOW()),
('eu-00000000000000000000000000000000',  (SELECT id FROM region WHERE name='Europe'), '34.88.82.214', 'active', (SELECT id FROM pool WHERE tag='geonodeeu'),9001, NOW()),
('asia-00000000000000000000000000000000',(SELECT id FROM region WHERE name='Asia'), '34.131.224.206', 'active', (SELECT id FROM pool WHERE tag='geonodeasia'),9002, NOW());

----------------------------------------------------------
-- 11. Worker Domains
----------------------------------------------------------
INSERT INTO worker_domains (worker_id, domain) VALUES
((SELECT id FROM worker WHERE name='usa-00000000000000000000000000000000'), 'geonode.usa.upstream-y.com'),
((SELECT id FROM worker WHERE name='eu-00000000000000000000000000000000'),  'netnut.eu.upstream-y.com'),
((SELECT id FROM worker WHERE name='asia-00000000000000000000000000000000'),'netnut.asia.upstream-y.com');

----------------------------------------------------------
-- Summary
----------------------------------------------------------
SELECT 'Database populated successfully!' AS status;

SELECT
    (SELECT COUNT(*) FROM region)   AS region_count,
    (SELECT COUNT(*) FROM country)  AS country_count,
    (SELECT COUNT(*) FROM "user")   AS user_count,
    (SELECT COUNT(*) FROM pool)     AS pool_count,
    (SELECT COUNT(*) FROM upstream) AS upstream_count,
    (SELECT COUNT(*) FROM worker)   AS worker_count;

COMMIT;
