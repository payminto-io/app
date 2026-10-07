INSERT INTO currencies (name, code, type, visible, deposit_enabled, withdrawal_enabled, wallet_precision, base_precision, created_at, updated_at) VALUES
  ('Bitcoin',  'BTC',  'native', true, true, true, 8,  8,  NOW(), NOW()),
  ('Ethereum', 'ETH',  'native', true, true, true, 18, 18, NOW(), NOW()),
  ('Tether',   'USDT', 'token',  true, true, true, 6,  6,  NOW(), NOW()),
  ('USD Coin', 'USDC', 'token',  true, true, true, 6,  6,  NOW(), NOW()),
  ('Tron',     'TRX',  'native', true, true, true, 6,  6,  NOW(), NOW()),
  ('Polygon',  'POL',  'native', true, true, true, 18, 18, NOW(), NOW())
ON CONFLICT (code) DO NOTHING;

-- Testnet token contracts. These are well-known testnet faucet/mock contracts.
-- USDC on Sepolia (Circle's testnet)
INSERT INTO blockchain_currencies (blockchain_id, currency_id, address, standard, currency_code, blockchain_code, deposit_enabled, withdrawal_enabled, visible, wallet_precision, created_at, updated_at) VALUES
  ((SELECT id FROM blockchains WHERE code='ETH'),     (SELECT id FROM currencies WHERE code='ETH'),  '',                                            'native', 'ETH',  'ETH',     true, true, true, 18, NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='ETH'),     (SELECT id FROM currencies WHERE code='USDC'), '0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238', 'ERC20',  'USDC', 'ETH',     true, true, true, 6,  NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='BASE'),    (SELECT id FROM currencies WHERE code='ETH'),  '',                                            'native', 'ETH',  'BASE',    true, true, true, 18, NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='BASE'),    (SELECT id FROM currencies WHERE code='USDC'), '0x036CbD53842c5426634e7929541eC2318f3dCF7e', 'ERC20',  'USDC', 'BASE',    true, true, true, 6,  NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='POLYGON'), (SELECT id FROM currencies WHERE code='POL'),  '',                                            'native', 'POL',  'POLYGON', true, true, true, 18, NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='POLYGON'), (SELECT id FROM currencies WHERE code='USDC'), '0x41E94Eb019C0762f9Bfcf9Fb1E58725BfB0e7582', 'ERC20',  'USDC', 'POLYGON', true, true, true, 6,  NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='BTC'),     (SELECT id FROM currencies WHERE code='BTC'),  '',                                            'native', 'BTC',  'BTC',     true, true, true, 8,  NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='TRX'),     (SELECT id FROM currencies WHERE code='TRX'),  '',                                            'native', 'TRX',  'TRX',     true, true, true, 6,  NOW(), NOW())
ON CONFLICT DO NOTHING;
