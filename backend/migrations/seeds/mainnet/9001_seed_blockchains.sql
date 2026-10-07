INSERT INTO blockchain_families (name, code, created_at, updated_at) VALUES
  ('EVM', 'evm', NOW(), NOW()),
  ('Bitcoin', 'btc', NOW(), NOW()),
  ('Tron', 'trx', NOW(), NOW())
ON CONFLICT (code) DO NOTHING;

INSERT INTO blockchains (code, name, blockchain_family_id, min_confirmations, height, status, explorer_address, explorer_tx, chain_id, created_at, updated_at) VALUES
  ('ETH',     'Ethereum',      (SELECT id FROM blockchain_families WHERE code='evm'), 12,  0, 'active', 'https://etherscan.io/address/',           'https://etherscan.io/tx/',           1,    NOW(), NOW()),
  ('BASE',    'Base',          (SELECT id FROM blockchain_families WHERE code='evm'), 12,  0, 'active', 'https://basescan.org/address/',           'https://basescan.org/tx/',           8453, NOW(), NOW()),
  ('POLYGON', 'Polygon',       (SELECT id FROM blockchain_families WHERE code='evm'), 128, 0, 'active', 'https://polygonscan.com/address/',        'https://polygonscan.com/tx/',        137,  NOW(), NOW()),
  ('BTC',     'Bitcoin',       (SELECT id FROM blockchain_families WHERE code='btc'), 3,   0, 'active', 'https://mempool.space/address/',          'https://mempool.space/tx/',          NULL, NOW(), NOW()),
  ('TRX',     'Tron',          (SELECT id FROM blockchain_families WHERE code='trx'), 19,  0, 'active', 'https://tronscan.org/#/address/',         'https://tronscan.org/#/transaction/', NULL, NOW(), NOW())
ON CONFLICT (code) DO NOTHING;

-- Solana (ticket 09). family is the label the adapter registry and key resolver switch on.
INSERT INTO blockchain_families (name, code, family, path, supports_hd_wallet, created_at, updated_at) VALUES
  ('Solana', 'sol', 'SOL_Family', 'm/44''/501''/%d''/0''', true, NOW(), NOW())
ON CONFLICT (code) DO NOTHING;

-- min_confirmations 32: a Solana deposit is credited only once finalized (rooted), never on confirmed.
INSERT INTO blockchains (code, name, family, client, blockchain_family_id, min_confirmations, height, status, explorer_address, explorer_tx, chain_id, created_at, updated_at) VALUES
  ('SOLANA', 'Solana', 'SOL_Family', 'solana-rpc', (SELECT id FROM blockchain_families WHERE code='sol'), 32, 0, 'active', 'https://explorer.solana.com/address/', 'https://explorer.solana.com/tx/', NULL, NOW(), NOW())
ON CONFLICT (code) DO NOTHING;
UPDATE blockchains SET family = 'SOL_Family' WHERE code = 'SOLANA' AND (family IS NULL OR family = '');
