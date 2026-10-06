INSERT INTO configurations (key, value, description, category, created_at, updated_at) VALUES
  ('mode',                            'mainnet', 'Network mode stamp',          'system',   NOW(), NOW()),
  ('sweep.eth.threshold',             '0.005',   'Min ETH balance to sweep',    'sweep',    NOW(), NOW()),
  ('sweep.btc.threshold',             '0.001',   'Min BTC balance to sweep',    'sweep',    NOW(), NOW()),
  ('sweep.poll_interval_seconds',     '120',     'Sweep processor tick',         'sweep',    NOW(), NOW()),
  ('block_processor.eth.poll_seconds','5',       'ETH block poll interval',      'workers',  NOW(), NOW()),
  ('block_processor.btc.poll_seconds','60',      'BTC block poll interval',      'workers',  NOW(), NOW()),
  ('webhook.retry_max_attempts',      '7',       'Max webhook retry attempts',   'webhooks', NOW(), NOW()),
  ('payment.expiry_minutes',          '30',      'Payment request expiry',       'payments', NOW(), NOW())
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();
