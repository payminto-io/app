-- Testnet blockchains
INSERT INTO blockchain_families (name, code, created_at, updated_at) VALUES
  ('EVM', 'evm', NOW(), NOW()),
  ('Bitcoin', 'btc', NOW(), NOW()),
  ('Tron', 'trx', NOW(), NOW())
ON CONFLICT (code) DO NOTHING;

INSERT INTO blockchains (code, name, blockchain_family_id, min_confirmations, height, status, explorer_address, explorer_tx, chain_id, created_at, updated_at) VALUES
  ('ETH',     'Ethereum Sepolia', (SELECT id FROM blockchain_families WHERE code='evm'), 12,  0, 'active', 'https://sepolia.etherscan.io/address/',     'https://sepolia.etherscan.io/tx/',     11155111, NOW(), NOW()),
  ('BASE',    'Base Sepolia',     (SELECT id FROM blockchain_families WHERE code='evm'), 12,  0, 'active', 'https://sepolia.basescan.org/address/',     'https://sepolia.basescan.org/tx/',     84532,    NOW(), NOW()),
  ('POLYGON', 'Polygon Amoy',     (SELECT id FROM blockchain_families WHERE code='evm'), 128, 0, 'active', 'https://amoy.polygonscan.com/address/',     'https://amoy.polygonscan.com/tx/',     80002,    NOW(), NOW()),
  ('BTC',     'Bitcoin Testnet3', (SELECT id FROM blockchain_families WHERE code='btc'), 3,   0, 'active', 'https://blockstream.info/testnet/address/', 'https://blockstream.info/testnet/tx/', NULL,     NOW(), NOW()),
  ('TRX',     'Tron Nile',        (SELECT id FROM blockchain_families WHERE code='trx'), 19,  0, 'active', 'https://nile.tronscan.org/#/address/',      'https://nile.tronscan.org/#/transaction/', NULL, NOW(), NOW())
ON CONFLICT (code) DO NOTHING;
