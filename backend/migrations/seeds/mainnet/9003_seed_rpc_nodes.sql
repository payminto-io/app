-- Mainnet RPC endpoints. Replace ${API_KEY} placeholders via env var
-- substitution before applying. Free public nodes are listed first as
-- failover; paid Alchemy/Infura should be added by ops.
INSERT INTO rpc_nodes (blockchain_id, name, url, weight, priority, status, created_at, updated_at) VALUES
  ((SELECT id FROM blockchains WHERE code='ETH'),     'PublicNode',  'https://ethereum-rpc.publicnode.com',  1, 10, 'healthy', NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='ETH'),     'LlamaRPC',    'https://eth.llamarpc.com',              1, 20, 'healthy', NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='ETH'),     'dRPC',        'https://eth.drpc.org',                  1, 30, 'healthy', NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='ETH'),     '1RPC',        'https://1rpc.io/eth',                   1, 40, 'healthy', NOW(), NOW()),

  ((SELECT id FROM blockchains WHERE code='BASE'),    'PublicNode',  'https://base-rpc.publicnode.com',       1, 10, 'healthy', NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='BASE'),    'Official',    'https://mainnet.base.org',              1, 20, 'healthy', NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='BASE'),    'LlamaRPC',    'https://base.llamarpc.com',             1, 30, 'healthy', NOW(), NOW()),

  ((SELECT id FROM blockchains WHERE code='POLYGON'), 'PublicNode',  'https://polygon-bor-rpc.publicnode.com',1, 10, 'healthy', NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='POLYGON'), 'LlamaRPC',    'https://polygon.llamarpc.com',          1, 20, 'healthy', NOW(), NOW()),

  ((SELECT id FROM blockchains WHERE code='BTC'),     'Blockstream', 'https://blockstream.info/api',          1, 10, 'healthy', NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='BTC'),     'Mempool',     'https://mempool.space/api',             1, 20, 'healthy', NOW(), NOW()),

  ((SELECT id FROM blockchains WHERE code='TRX'),     'TronGrid',    'https://api.trongrid.io',               1, 10, 'healthy', NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='TRX'),     'NowNodes',    'https://trx.nownodes.io',               1, 20, 'healthy', NOW(), NOW())
ON CONFLICT DO NOTHING;

-- Solana: the public endpoint is rate limited; ops should add a paid node (Helius, Triton, QuickNode) at a lower priority number.
INSERT INTO rpc_nodes (blockchain_id, name, url, weight, priority, status, created_at, updated_at) VALUES
  ((SELECT id FROM blockchains WHERE code='SOLANA'), 'Solana-Public', 'https://api.mainnet-beta.solana.com', 1, 50, 'healthy', NOW(), NOW())
ON CONFLICT DO NOTHING;
