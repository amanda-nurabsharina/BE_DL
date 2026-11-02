-- Dental Laboratory Management System (DLMS)
-- Migration: 000002_seed_data.sql
-- Default Master Data Seed

-- ============================================================================
-- 1. SEED ROLES
-- ============================================================================
INSERT INTO roles (name, description) VALUES
('OWNER', 'Pemilik Laboratorium: Full akses bisnis, persetujuan pembelian, laporan keuangan & user management'),
('ADMIN', 'Admin / Asisten Lab: Input order dokter, atur jadwal, koordinasi teknisi, kirim notifikasi WA'),
('TECHNICIAN', 'Teknisi Gigi: Mengerjakan tahapan produksi, catat konsumsi bahan, update status pengerjaan'),
('CASHIER', 'Kasir / Finance: Cetak invoice, input pembayaran DP/pelunasan, catat pengeluaran operasional')
ON CONFLICT (name) DO NOTHING;

-- ============================================================================
-- 2. SEED UNITS OF MEASURE
-- ============================================================================
INSERT INTO units_of_measure (code, name) VALUES
('pcs', 'Pieces / Buah'),
('gr', 'Gram'),
('kg', 'Kilogram'),
('ml', 'Mililiter'),
('box', 'Kotak / Box'),
('tube', 'Tube'),
('set', 'Set')
ON CONFLICT (code) DO NOTHING;

-- ============================================================================
-- 3. SEED DENTURE TYPES & PRODUCTION STAGE TEMPLATES
-- ============================================================================
-- 1. Zirconia All-Ceramic Crown
INSERT INTO denture_types (code, name, category, base_price, estimated_days, description) VALUES
('ZIRCONIA_CROWN', 'Crown / Bridge All-Ceramic Zirconia', 'FIXED_PROSTHESIS', 750000, 3, 'Mahkota gigi tiruan penuh Zirconia multi-layer kuat dan estetis')
ON CONFLICT (code) DO NOTHING;

-- 2. Porcelain Fused to Metal (PFM)
INSERT INTO denture_types (code, name, category, base_price, estimated_days, description) VALUES
('PFM_CROWN', 'Crown / Bridge PFM (Porcelain Fused to Metal)', 'FIXED_PROSTHESIS', 450000, 4, 'Mahkota gigi dengan koping logam dilapisi porselen keramik')
ON CONFLICT (code) DO NOTHING;

-- 3. Acrylic Removable Denture
INSERT INTO denture_types (code, name, category, base_price, estimated_days, description) VALUES
('ACRYLIC_DENTURE', 'Gigi Tiruan Akrilik (Lepasan)', 'REMOVABLE_DENTURE', 300000, 5, 'Plat akrilik lepasan sebagian atau penuh')
ON CONFLICT (code) DO NOTHING;

-- 4. Valplast Flexible Denture
INSERT INTO denture_types (code, name, category, base_price, estimated_days, description) VALUES
('VALPLAST_FLEXIBLE', 'Gigi Tiruan Fleksibel (Valplast)', 'REMOVABLE_DENTURE', 550000, 4, 'Gigi tiruan nilon termoplastik lentur tanpa kawat logam')
ON CONFLICT (code) DO NOTHING;

-- 5. Orthodontic Retainer
INSERT INTO denture_types (code, name, category, base_price, estimated_days, description) VALUES
('ORTHO_RETAINER', 'Retainer Hawley / Clear Orthodontic', 'ORTHODONTIC', 250000, 2, 'Alat retensi ortodonti kawat Hawley atau transparan')
ON CONFLICT (code) DO NOTHING;

-- Insert Stage Templates for Zirconia Crown
DO $$
DECLARE
    zirconia_id INT;
BEGIN
    SELECT id INTO zirconia_id FROM denture_types WHERE code = 'ZIRCONIA_CROWN';
    IF zirconia_id IS NOT NULL THEN
        INSERT INTO production_stage_templates (denture_type_id, stage_order, stage_code, stage_name, description) VALUES
        (zirconia_id, 1, 'CAST_MODEL', 'Cetak Model Gips & Die Preparation', 'Pengecoran gips tipe 4 dan preparasi die model kerja'),
        (zirconia_id, 2, 'CAD_SCAN_DESIGN', '3D Scanning & CAD Design', 'Scanning model dan perancangan anatomi mahkota gigi'),
        (zirconia_id, 3, 'CAM_MILLING', 'CAM Milling Zirconia Block', 'Proses milling disk Zirconia pada mesin CNC'),
        (zirconia_id, 4, 'SINTERING', 'Sintering Oven', 'Pembakaran kristalisasi Zirconia suhu tinggi 1500°C'),
        (zirconia_id, 5, 'GLAZE_POLISH', 'Staining, Glazing & Polishing', 'Pewarnaan shade dan finishing kilap permukaan'),
        (zirconia_id, 6, 'QC_REVIEW', 'Quality Control & Fit Test', 'Pengecekan kontak proksimal, oklusi, marginal fit dan warna')
        ON CONFLICT DO NOTHING;
    END IF;
END $$;

-- Insert Stage Templates for PFM Crown
DO $$
DECLARE
    pfm_id INT;
BEGIN
    SELECT id INTO pfm_id FROM denture_types WHERE code = 'PFM_CROWN';
    IF pfm_id IS NOT NULL THEN
        INSERT INTO production_stage_templates (denture_type_id, stage_order, stage_code, stage_name, description) VALUES
        (pfm_id, 1, 'CAST_MODEL', 'Cetak Model Gips & Die', 'Pengecoran gips tipe 4 dan preparasi die model kerja'),
        (pfm_id, 2, 'WAX_COPING', 'Wax-up Koping Logam', 'Pembuatan pola lilin untuk koping logam'),
        (pfm_id, 3, 'METAL_CASTING', 'Casting & Finishing Logam', 'Pengecoran logam nikel/kobalt krom dan finishing koping'),
        (pfm_id, 4, 'CERAMIC_LAYERING', 'Aplikasi & Pembakaran Porselen', 'Pelapisan opaque, dentin, enamel dan bake keramik'),
        (pfm_id, 5, 'GLAZE_POLISH', 'Glaze & Final Polish', 'Poles akhir dan kilap porselen'),
        (pfm_id, 6, 'QC_REVIEW', 'Quality Control', 'Pemeriksaan oklusi, adaptasi margin dan kecocokan shade guide')
        ON CONFLICT DO NOTHING;
    END IF;
END $$;

-- Insert Stage Templates for Acrylic Denture
DO $$
DECLARE
    acrylic_id INT;
BEGIN
    SELECT id INTO acrylic_id FROM denture_types WHERE code = 'ACRYLIC_DENTURE';
    IF acrylic_id IS NOT NULL THEN
        INSERT INTO production_stage_templates (denture_type_id, stage_order, stage_code, stage_name, description) VALUES
        (acrylic_id, 1, 'CAST_MODEL', 'Cetak Model Kerja Gips', 'Pengecoran gips biru keras dan pembuatan base model'),
        (acrylic_id, 2, 'BITE_RIM', 'Pembuatan Galangan Gigit & Okludator', 'Pembuatan wax bite rim dan mounting artikulator'),
        (acrylic_id, 3, 'TOOTH_SETUP', 'Penyusunan Elemen Gigi Tiruan', 'Penyusunan gigi artifisial sesuai oklusi anatomis'),
        (acrylic_id, 4, 'FLASKING_CURING', 'Kuvet & Polimerisasi Akrilik', 'Proses packing akrilik heat cure dan boiling'),
        (acrylic_id, 5, 'TRIM_POLISH', 'Deflasking, Trimming & Polishing', 'Pembongkaran kuvet, pengasahan sayap dan poles kilap'),
        (acrylic_id, 6, 'QC_REVIEW', 'Quality Control', 'Cek ketebalan plat, retensi, stabilitas dan kebersihan plat')
        ON CONFLICT DO NOTHING;
    END IF;
END $$;

-- Insert Stage Templates for Valplast
DO $$
DECLARE
    valplast_id INT;
BEGIN
    SELECT id INTO valplast_id FROM denture_types WHERE code = 'VALPLAST_FLEXIBLE';
    IF valplast_id IS NOT NULL THEN
        INSERT INTO production_stage_templates (denture_type_id, stage_order, stage_code, stage_name, description) VALUES
        (valplast_id, 1, 'CAST_MODEL', 'Cetak Model Gips Master', 'Pengecoran model master dan duplikasi model investmen'),
        (valplast_id, 2, 'DESIGN_WAX', 'Survey, Block-out & Waxing', 'Penentuan arah pasang, klamer nilon dan pola lilin'),
        (valplast_id, 3, 'INJECTION_MOLDING', 'Injeksi Valplast Termoplastik', 'Pemanasan cartridge nilon dan injeksi presisi'),
        (valplast_id, 4, 'TRIM_POLISH', 'Trimming & Special Polishing', 'Pengasahan bur karet khusus nilon dan pemolesan kilap'),
        (valplast_id, 5, 'QC_REVIEW', 'Quality Control', 'Cek kelenturan klamer, adaptasi tepi dan retensi')
        ON CONFLICT DO NOTHING;
    END IF;
END $$;

-- Insert Stage Templates for Ortho Retainer
DO $$
DECLARE
    ortho_id INT;
BEGIN
    SELECT id INTO ortho_id FROM denture_types WHERE code = 'ORTHO_RETAINER';
    IF ortho_id IS NOT NULL THEN
        INSERT INTO production_stage_templates (denture_type_id, stage_order, stage_code, stage_name, description) VALUES
        (ortho_id, 1, 'CAST_MODEL', 'Cetak Model Kerja', 'Pengecoran gips putih/kuning rapi'),
        (ortho_id, 2, 'WIRE_BENDING', 'Pembengkokan Kawat Labial Bow & Klamer', 'Fabrikasi kawat stainless steel sesuai anatomi gigi'),
        (ortho_id, 3, 'ACRYLIC_CURE', 'Aplikasi Akrilik & Polimerisasi', 'Metode sprinkle-on akrilik self/cold cure'),
        (ortho_id, 4, 'TRIM_POLISH', 'Trimming & Pemolesan Halus', 'Penghalusan tepi palatal/lingual agar nyaman di mulut'),
        (ortho_id, 5, 'QC_REVIEW', 'Quality Control', 'Uji pasif kawat dan kenyamanan tepi plat')
        ON CONFLICT DO NOTHING;
    END IF;
END $$;

-- ============================================================================
-- 4. SEED EXPENSE CATEGORIES
-- ============================================================================
INSERT INTO expense_categories (code, name) VALUES
('BAHAN_BAKU', 'Pembelian Bahan Baku & Logam Gigi'),
('ALAT_BUR', 'Mata Bur, Instrumen & Perlengkapan Lab'),
('UTILITAS', 'Listrik, Air & Gas Laboratorium'),
('OPERASIONAL', 'Kemasan, ATK & Transportasi Kurir'),
('GAJI_HONOR', 'Honor Teknisi & Pegawai Lab'),
('MAINTENANCE', 'Servis & Perawatan Mesin / Furnace'),
('LAIN_LAIN', 'Biaya Operasional Lain-lain')
ON CONFLICT (code) DO NOTHING;

-- ============================================================================
-- 5. SEED INITIAL DEFAULT USERS (Password: admin12345)
-- Bcrypt Hash of "admin12345": $2a$10$iMfv0M0s1U5XbS1iJq2VSu3uJb7a0d4tN7A2A9P3FzK8P8W8B8S2m
-- ============================================================================
DO $$
DECLARE
    owner_role_id INT;
    admin_role_id INT;
    tech_role_id INT;
    cashier_role_id INT;
    default_pw VARCHAR(255) := '$2a$10$wT282hH5F3p1iZ3Y5wK0vOB/9v9rXgNqVnJn0w6B8XgV8zN8uG6pS'; -- admin12345
BEGIN
    SELECT id INTO owner_role_id FROM roles WHERE name = 'OWNER';
    SELECT id INTO admin_role_id FROM roles WHERE name = 'ADMIN';
    SELECT id INTO tech_role_id FROM roles WHERE name = 'TECHNICIAN';
    SELECT id INTO cashier_role_id FROM roles WHERE name = 'CASHIER';

    -- Default Owner Account
    INSERT INTO users (username, email, password_hash, full_name, phone, role_id)
    VALUES ('owner', 'owner@dentallab.local', default_pw, 'Owner Dental Lab', '081234567890', owner_role_id)
    ON CONFLICT (username) DO NOTHING;

    -- Default Admin Account
    INSERT INTO users (username, email, password_hash, full_name, phone, role_id)
    VALUES ('admin', 'admin@dentallab.local', default_pw, 'Asisten Lab / Admin', '081234567891', admin_role_id)
    ON CONFLICT (username) DO NOTHING;

    -- Default Technician Account
    INSERT INTO users (username, email, password_hash, full_name, phone, role_id)
    VALUES ('teknisi1', 'teknisi@dentallab.local', default_pw, 'Teknisi Utama Gigi', '081234567892', tech_role_id)
    ON CONFLICT (username) DO NOTHING;

    -- Default Cashier Account
    INSERT INTO users (username, email, password_hash, full_name, phone, role_id)
    VALUES ('kasir', 'kasir@dentallab.local', default_pw, 'Kasir / Finance', '081234567893', cashier_role_id)
    ON CONFLICT (username) DO NOTHING;
END $$;
