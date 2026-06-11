-- Initial database schema for SkyFee

CREATE TABLE IF NOT EXISTS schools (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    paybill VARCHAR(50) NOT NULL,
    account_number VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS students (
    id SERIAL PRIMARY KEY,
    school_id INT REFERENCES schools(id) ON DELETE CASCADE,
    admission_number VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    grade VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_school_admission UNIQUE(school_id, admission_number)
);

CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY,
    school_id INT REFERENCES schools(id),
    student_admission_number VARCHAR(100) NOT NULL,
    student_name VARCHAR(255) NOT NULL,
    parent_name VARCHAR(255) NOT NULL,
    amount_kes NUMERIC(10, 2) NOT NULL,
    amount_sats BIGINT NOT NULL,
    lightning_invoice TEXT NOT NULL,
    payment_hash VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    mpesa_receipt VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Seed some sample schools and students for hackathon testing
INSERT INTO schools (id, name, paybill, account_number) VALUES 
(1, 'Alliance High School', '222111', 'SchoolFees'),
(2, 'Kenya High School', '333222', 'SchoolFees'),
(3, 'Lenana School', '444333', 'SchoolFees')
ON CONFLICT (id) DO NOTHING;

INSERT INTO students (id, school_id, admission_number, name, grade) VALUES 
(1, 1, 'AHS-8899', 'John Kiprop', 'Form 3 Green'),
(2, 1, 'AHS-9012', 'David Mwangi', 'Form 1 Blue'),
(3, 2, 'KHS-4455', 'Sarah Cherono', 'Form 4 East'),
(4, 3, 'LEN-1234', 'Joseph Kamau', 'Form 2 West')
ON CONFLICT (id) DO NOTHING;
