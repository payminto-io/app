INSERT INTO currencies (name, code, type, visible, deposit_enabled, withdrawal_enabled, wallet_precision, base_precision, created_at, updated_at) VALUES
  ('Bitcoin',  'BTC',  'native', true, true, true, 8,  8,  NOW(), NOW()),
  ('Ethereum', 'ETH',  'native', true, true, true, 18, 18, NOW(), NOW()),
  ('Tether',   'USDT', 'token',  true, true, true, 6,  6,  NOW(), NOW()),
  ('USD Coin', 'USDC', 'token',  true, true, true, 6,  6,  NOW(), NOW()),
  ('Tron',     'TRX',  'native', true, true, true, 6,  6,  NOW(), NOW()),
  ('Polygon',  'POL',  'native', true, true, true, 18, 18, NOW(), NOW())
ON CONFLICT (code) DO NOTHING;

-- REAL mainnet contract addresses
INSERT INTO blockchain_currencies (blockchain_id, currency_id, address, standard, currency_code, blockchain_code, deposit_enabled, withdrawal_enabled, visible, wallet_precision, created_at, updated_at) VALUES
  ((SELECT id FROM blockchains WHERE code='ETH'),     (SELECT id FROM currencies WHERE code='ETH'),  '',                                            'native', 'ETH',  'ETH',     true, true, true, 18, NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='ETH'),     (SELECT id FROM currencies WHERE code='USDT'), '0xdAC17F958D2ee523a2206206994597C13D831ec7', 'ERC20',  'USDT', 'ETH',     true, true, true, 6,  NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='ETH'),     (SELECT id FROM currencies WHERE code='USDC'), '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48', 'ERC20',  'USDC', 'ETH',     true, true, true, 6,  NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='BASE'),    (SELECT id FROM currencies WHERE code='ETH'),  '',                                            'native', 'ETH',  'BASE',    true, true, true, 18, NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='BASE'),    (SELECT id FROM currencies WHERE code='USDC'), '0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913', 'ERC20',  'USDC', 'BASE',    true, true, true, 6,  NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='POLYGON'), (SELECT id FROM currencies WHERE code='USDT'), '0xc2132D05D31c914a87C6611C10748AEb04B58e8F', 'ERC20',  'USDT', 'POLYGON', true, true, true, 6,  NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='POLYGON'), (SELECT id FROM currencies WHERE code='POL'),  '',                                            'native', 'POL',  'POLYGON', true, true, true, 18, NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='POLYGON'), (SELECT id FROM currencies WHERE code='USDC'), '0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359', 'ERC20',  'USDC', 'POLYGON', true, true, true, 6,  NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='BTC'),     (SELECT id FROM currencies WHERE code='BTC'),  '',                                            'native', 'BTC',  'BTC',     true, true, true, 8,  NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='TRX'),     (SELECT id FROM currencies WHERE code='TRX'),  '',                                            'native', 'TRX',  'TRX',     true, true, true, 6,  NOW(), NOW()),
  ((SELECT id FROM blockchains WHERE code='TRX'),     (SELECT id FROM currencies WHERE code='USDT'), 'TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t',         'TRC20',  'USDT', 'TRX',     true, true, true, 6,  NOW(), NOW())
ON CONFLICT DO NOTHING;
