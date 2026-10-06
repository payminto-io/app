-- Public testnet RPC endpoints — no API keys required
INSERT INTO rpc_nodes (blockchain_id, name, url, weight, priority, status, created_at, updated_at) VALUES
  ((SELECT id FROM blockchains WHERE code='ETH'),     'PublicNode-Sepolia',    'https://ethereum-sepolia-rpc.publicnode.com', 1, 10, 'healthy', NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='ETH'),     'dRPC-Sepolia',          'https://sepolia.drpc.org',                    1, 20, 'healthy', NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='BASE'),    'PublicNode-BaseSep',    'https://base-sepolia-rpc.publicnode.com',     1, 10, 'healthy', NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='BASE'),    'Official-BaseSepolia',  'https://sepolia.base.org',                    1, 20, 'healthy', NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='POLYGON'), 'PublicNode-Amoy',       'https://polygon-amoy-rpc.publicnode.com',     1, 10, 'healthy', NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='BTC'),     'Blockstream-Testnet',   'https://blockstream.info/testnet/api',        1, 10, 'healthy', NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='TRX'),     'TronGrid-Nile',         'https://nile.trongrid.io',                    1, 10, 'healthy', NOW(), NOW())
ON CONFLICT DO NOTHING;
