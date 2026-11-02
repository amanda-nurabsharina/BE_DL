-- Migration: 000004_inventory_master_and_seed.sql
-- Enhance items table with supplier, lot_number, and expiry_date for master data tracking

ALTER TABLE items ADD COLUMN IF NOT EXISTS supplier_id UUID REFERENCES suppliers(id) ON DELETE SET NULL;
ALTER TABLE items ADD COLUMN IF NOT EXISTS lot_number VARCHAR(100);
ALTER TABLE items ADD COLUMN IF NOT EXISTS expiry_date DATE;

