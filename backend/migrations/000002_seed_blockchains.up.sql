INSERT INTO blockchain_families (name, code, created_at, updated_at) VALUES
  ('EVM', 'evm', NOW(), NOW()),
  ('Bitcoin', 'btc', NOW(), NOW()),
  ('Tron', 'trx', NOW(), NOW());

INSERT INTO blockchains (code, name, blockchain_family_id, min_confirmations, height, status, explorer_tx, explorer_address, created_at, updated_at) VALUES
  ('ETH', 'Ethereum', 1, 12, 0, 'active', 'https://etherscan.io/tx/', 'https://etherscan.io/address/', NOW(), NOW()),
  ('BASE', 'Base', 1, 12, 0, 'active', 'https://basescan.org/tx/', 'https://basescan.org/address/', NOW(), NOW()),
  ('POLYGON', 'Polygon', 1, 128, 0, 'active', 'https://polygonscan.com/tx/', 'https://polygonscan.com/address/', NOW(), NOW()),
  ('BTC', 'Bitcoin', 2, 3, 0, 'active', 'https://mempool.space/tx/', 'https://mempool.space/address/', NOW(), NOW()),
  ('TRX', 'Tron', 3, 19, 0, 'active', 'https://tronscan.org/#/transaction/', 'https://tronscan.org/#/address/', NOW(), NOW());

INSERT INTO currencies (name, code, type, visible, deposit_enabled, withdrawal_enabled, wallet_precision, base_precision, created_at, updated_at) VALUES
  ('Bitcoin', 'BTC', 'native', true, true, true, 8, 8, NOW(), NOW()),
  ('Ethereum', 'ETH', 'native', true, true, true, 18, 18, NOW(), NOW()),
  ('Tether', 'USDT', 'token', true, true, true, 6, 6, NOW(), NOW()),
  ('USD Coin', 'USDC', 'token', true, true, true, 6, 6, NOW(), NOW()),
  ('Tron', 'TRX', 'native', true, true, true, 6, 6, NOW(), NOW());
