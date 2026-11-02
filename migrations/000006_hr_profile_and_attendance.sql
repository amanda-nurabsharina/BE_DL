-- Migration: 000006_hr_profile_and_attendance.sql
-- User Profile fields (avatar, skills, schedule, bio) & Staff Attendance / Overtime table

ALTER TABLE users ADD COLUMN IF NOT EXISTS employee_code VARCHAR(50) DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url VARCHAR(255) DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS specialization VARCHAR(255) DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS skills VARCHAR(255) DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS work_schedule VARCHAR(150) DEFAULT 'Senin - Sabtu (08:00 - 17:00)';
ALTER TABLE users ADD COLUMN IF NOT EXISTS address TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS join_date VARCHAR(30) DEFAULT '2024-01-15';
ALTER TABLE users ADD COLUMN IF NOT EXISTS bio TEXT DEFAULT '';

CREATE TABLE IF NOT EXISTS staff_attendances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    attendance_date DATE NOT NULL DEFAULT CURRENT_DATE,
    shift_schedule VARCHAR(100) DEFAULT 'Pagi (08:00 - 17:00)',
    check_in VARCHAR(20) DEFAULT '08:00',
    check_out VARCHAR(20) DEFAULT '17:00',
    status VARCHAR(30) DEFAULT 'HADIR', -- HADIR, TERLAMBAT, IZIN, SAKIT, CUTI
    overtime_hours NUMERIC(5,2) DEFAULT 0,
    overtime_notes TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

