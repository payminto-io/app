INSERT INTO permissions (name, display_name, description, created_at, updated_at) VALUES
  ('payments:read',     'Read Payments',     'View payment requests',      NOW(), NOW()),
  ('payments:create',   'Create Payments',   'Create new payment requests', NOW(), NOW()),
  ('wallets:read',      'Read Wallets',      'View wallet balances',        NOW(), NOW()),
  ('wallets:manage',    'Manage Wallets',    'Configure wallets and keys',  NOW(), NOW()),
  ('sweeps:read',       'Read Sweeps',       'View sweep history',          NOW(), NOW()),
  ('sweeps:execute',    'Execute Sweeps',    'Trigger manual sweeps',       NOW(), NOW()),
  ('withdrawals:read',  'Read Withdrawals',  'View payouts',                NOW(), NOW()),
  ('withdrawals:create','Create Withdrawals','Initiate payouts',            NOW(), NOW()),
  ('withdrawals:approve','Approve Withdrawals','Approve held payouts',       NOW(), NOW()),
  ('webhooks:manage',   'Manage Webhooks',   'CRUD webhooks',               NOW(), NOW()),
  ('analytics:read',    'Read Analytics',    'View dashboards',             NOW(), NOW()),
  ('members:manage',    'Manage Members',    'Invite/remove team members',  NOW(), NOW()),
  ('system:admin',      'System Admin',      'Restart workers, edit config', NOW(), NOW())
ON CONFLICT (name) DO NOTHING;

INSERT INTO roles (name, display_name, description, created_at, updated_at) VALUES
  ('owner',                  'Owner',                  'Full access; cannot be removed',          NOW(), NOW()),
  ('admin',                  'Admin',                  'Manage all projects and team',            NOW(), NOW()),
  ('project_lead',           'Project Lead',           'Create/update projects, view analytics',  NOW(), NOW()),
  ('project_manager',        'Project Manager',        'View-only with export',                   NOW(), NOW()),
  ('project_ops',            'Project Ops',            'View payment and customer data',          NOW(), NOW()),
  ('platform_referral_admin','Platform Referral Admin','Manage referral campaigns',               NOW(), NOW())
ON CONFLICT (name) DO NOTHING;

-- Owner gets every permission
INSERT INTO role_permissions (role_id, permission_id)
SELECT (SELECT id FROM roles WHERE name='owner'), p.id FROM permissions p
ON CONFLICT DO NOTHING;

-- Admin gets all except system:admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT (SELECT id FROM roles WHERE name='admin'), p.id FROM permissions p WHERE p.name != 'system:admin'
ON CONFLICT DO NOTHING;

-- Project Lead: payments + wallets read/manage + sweeps + analytics
INSERT INTO role_permissions (role_id, permission_id)
SELECT (SELECT id FROM roles WHERE name='project_lead'), p.id FROM permissions p
WHERE p.name IN ('payments:read', 'payments:create', 'wallets:read', 'sweeps:read', 'analytics:read')
ON CONFLICT DO NOTHING;

-- Project Manager: read-only
INSERT INTO role_permissions (role_id, permission_id)
SELECT (SELECT id FROM roles WHERE name='project_manager'), p.id FROM permissions p
WHERE p.name IN ('payments:read', 'wallets:read', 'sweeps:read', 'withdrawals:read', 'analytics:read')
ON CONFLICT DO NOTHING;

-- Project Ops: payment + customer data only
INSERT INTO role_permissions (role_id, permission_id)
SELECT (SELECT id FROM roles WHERE name='project_ops'), p.id FROM permissions p
WHERE p.name IN ('payments:read', 'withdrawals:read')
ON CONFLICT DO NOTHING;
