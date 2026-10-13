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

-- Solana devnet (ticket 09). family is the label the adapter registry and key resolver switch on.
INSERT INTO blockchain_families (name, code, family, path, supports_hd_wallet, created_at, updated_at) VALUES
  ('Solana', 'sol', 'SOL_Family', 'm/44''/501''/%d''/0''', true, NOW(), NOW())
ON CONFLICT (code) DO NOTHING;

-- min_confirmations 32: a Solana deposit is credited only once finalized (rooted), never on confirmed.
INSERT INTO blockchains (code, name, family, client, blockchain_family_id, min_confirmations, height, status, explorer_address, explorer_tx, chain_id, created_at, updated_at) VALUES
  ('SOLANA', 'Solana Devnet', 'SOL_Family', 'solana-rpc', (SELECT id FROM blockchain_families WHERE code='sol'), 32, 0, 'active', 'https://explorer.solana.com/address/{address}?cluster=devnet', 'https://explorer.solana.com/tx/{tx}?cluster=devnet', NULL, NOW(), NOW())
ON CONFLICT (code) DO NOTHING;
UPDATE blockchains SET family = 'SOL_Family' WHERE code = 'SOLANA' AND (family IS NULL OR family = '');

-- Tempo Moderato testnet (chain 42431). Payments are TIP-20 transferWithMemo calls to one merchant
-- address, found from logs by the Tempo memo watcher (no adapter, no block processor, no RPC node row:
-- TEMPO_RPC_URL configures the endpoint). Finality is deterministic, so 1 confirmation is final.
INSERT INTO blockchain_families (name, code, family, created_at, updated_at) VALUES
  ('Tempo', 'tempo', 'TEMPO_Family', NOW(), NOW())
ON CONFLICT (code) DO NOTHING;

INSERT INTO blockchains (code, name, family, client, blockchain_family_id, min_confirmations, height, status, explorer_address, explorer_tx, chain_id, created_at, updated_at) VALUES
  ('TEMPO', 'Tempo Moderato', 'TEMPO_Family', 'tempo-logs', (SELECT id FROM blockchain_families WHERE code='tempo'), 1, 0, 'active', 'https://explore.testnet.tempo.xyz/address/', 'https://explore.testnet.tempo.xyz/tx/', 42431, NOW(), NOW())
ON CONFLICT (code) DO NOTHING;
