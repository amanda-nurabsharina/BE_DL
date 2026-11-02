-- Migration: 000005_patients_and_crm_loyalty.sql
-- Master Data Pasien & CRM Dokter/Klinik (Preferensi, Loyalty Diskon, History & Feedback)

ALTER TABLE customers ADD COLUMN IF NOT EXISTS favorite_shade VARCHAR(50) DEFAULT '';
ALTER TABLE customers ADD COLUMN IF NOT EXISTS favorite_denture_type VARCHAR(150) DEFAULT '';
ALTER TABLE customers ADD COLUMN IF NOT EXISTS loyalty_tier VARCHAR(30) DEFAULT 'REGULAR';
ALTER TABLE customers ADD COLUMN IF NOT EXISTS discount_percent NUMERIC(5,2) DEFAULT 0;

CREATE TABLE IF NOT EXISTS patients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(30) UNIQUE NOT NULL,
    customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
    name VARCHAR(150) NOT NULL,
    gender VARCHAR(10) DEFAULT 'L',
    age INT DEFAULT 0,
    phone VARCHAR(30) DEFAULT '',
    default_shade VARCHAR(50) DEFAULT 'VITA A2',
    last_teeth_numbers VARCHAR(100) DEFAULT '',
    allergies_or_notes TEXT DEFAULT '',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

ALTER TABLE production_orders ADD COLUMN IF NOT EXISTS patient_id UUID REFERENCES patients(id) ON DELETE SET NULL;
ALTER TABLE production_orders ADD COLUMN IF NOT EXISTS discount_percent NUMERIC(5,2) DEFAULT 0;
ALTER TABLE production_orders ADD COLUMN IF NOT EXISTS discount_amount NUMERIC(14,2) DEFAULT 0;
ALTER TABLE production_orders ADD COLUMN IF NOT EXISTS feedback_rating INT DEFAULT 0;
ALTER TABLE production_orders ADD COLUMN IF NOT EXISTS feedback_notes TEXT DEFAULT '';

