-- Migration: 000003_settings_and_superadmin.sql

-- 1. Add allowed_menus to roles table if not exists
ALTER TABLE roles ADD COLUMN IF NOT EXISTS allowed_menus JSONB DEFAULT '[]'::jsonb;

-- 2. Insert or update SUPER_ADMIN role and update existing roles
INSERT INTO roles (name, description, allowed_menus) VALUES
('SUPER_ADMIN', 'Super Admin: Full akses seluruh menu sistem, operasional & pengaturan (user, role, logo)', '["dashboard", "orders", "new_order", "inventory", "approvals", "finance", "customers", "settings"]'::jsonb)
ON CONFLICT (name) DO UPDATE SET 
    description = EXCLUDED.description,
    allowed_menus = EXCLUDED.allowed_menus;

UPDATE roles SET allowed_menus = '["dashboard", "orders", "new_order", "inventory", "approvals", "finance", "customers", "settings"]'::jsonb WHERE name = 'OWNER';
UPDATE roles SET allowed_menus = '["dashboard", "orders", "new_order", "inventory", "customers"]'::jsonb WHERE name = 'ADMIN';
UPDATE roles SET allowed_menus = '["orders"]'::jsonb WHERE name = 'TECHNICIAN';
UPDATE roles SET allowed_menus = '["dashboard", "finance", "orders"]'::jsonb WHERE name = 'CASHIER';

-- 3. Insert superadmin user (Password: admin12345)
DO $$
DECLARE
    super_admin_role_id INT;
    pw_hash VARCHAR(255) := '$2a$10$GVYQv2fTsdBKC3bvAXEHpuUHXq2MukjoNtj1zHsUG3IJikShnDU8K'; -- admin12345
BEGIN
    SELECT id INTO super_admin_role_id FROM roles WHERE name = 'SUPER_ADMIN';
    
    INSERT INTO users (username, email, password_hash, full_name, phone, role_id)
    VALUES ('superadmin', 'superadmin@dentallab.local', pw_hash, 'Super Admin Setup', '081199887766', super_admin_role_id)
    ON CONFLICT (username) DO UPDATE SET 
        role_id = super_admin_role_id,
        password_hash = pw_hash;
END $$;

-- 4. Create system_settings table
CREATE TABLE IF NOT EXISTS system_settings (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO system_settings (key, value) VALUES
('header_logo', ''),
('app_name', 'DentalLab Pro'),
('lab_name', 'Laboratorium Gigi Presisi')
ON CONFLICT (key) DO NOTHING;
