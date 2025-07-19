-- Migration: 001_create_tables.sql
-- Description: Create initial database schema for Scheduler system
-- Compatible with: PostgreSQL/Neon

-- Enable UUID extension for generating UUIDs
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Organization Table
CREATE TABLE organizations (
    organization_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. User Table
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'student', 'tutor')),
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    phone_number TEXT,
    email TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (org_id) REFERENCES organizations(organization_id) ON DELETE CASCADE
);

-- 3. Course Table
CREATE TABLE courses (
    course_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL,
    course_name TEXT NOT NULL,
    course_description TEXT,
    start_at TIMESTAMPTZ,
    end_at TIMESTAMPTZ,
    interval TEXT CHECK (interval IN ('week', 'bi-weekly', 'month')),
    frequency INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (org_id) REFERENCES organizations(organization_id) ON DELETE CASCADE,
    CHECK (end_at > start_at OR (end_at IS NULL OR start_at IS NULL))
);

-- 4. UserCourse Junction Table
CREATE TABLE user_courses (
    user_id UUID NOT NULL,
    course_id UUID NOT NULL,
    enrolled_at TIMESTAMPTZ DEFAULT NOW(),
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'completed', 'dropped')),
    PRIMARY KEY (user_id, course_id),
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    FOREIGN KEY (course_id) REFERENCES courses(course_id) ON DELETE CASCADE
);

-- 5. Availability Table
CREATE TABLE availability (
    availability_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL,
    user_id UUID NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'student', 'tutor')),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    matched BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (org_id) REFERENCES organizations(organization_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CHECK (end_time > start_time),
    CHECK (EXTRACT(EPOCH FROM (end_time - start_time)) >= 900) -- Minimum 15 minutes
);

-- 6. Class Table
CREATE TABLE classes (
    class_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    course_id UUID,
    org_id UUID NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    duration INTEGER NOT NULL CHECK (duration > 0), -- Duration in minutes
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (course_id) REFERENCES courses(course_id) ON DELETE SET NULL,
    FOREIGN KEY (org_id) REFERENCES organizations(organization_id) ON DELETE CASCADE
);

-- 7. ClassParticipant Table
CREATE TABLE class_participants (
    class_id UUID NOT NULL,
    user_id UUID NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('student', 'teacher')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (class_id, user_id),
    FOREIGN KEY (class_id) REFERENCES classes(class_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

-- 8. ClassAttendance Table
CREATE TABLE class_attendance (
    class_id UUID NOT NULL,
    user_id UUID NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('student', 'teacher')),
    attended BOOLEAN DEFAULT NULL, -- NULL = not yet determined
    notes TEXT,
    recorded_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (class_id, user_id),
    FOREIGN KEY (class_id) REFERENCES classes(class_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE
);

-- 9. Tracker Table
CREATE TABLE trackers (
    tracking_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    course_id UUID NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    required_classes INTEGER NOT NULL CHECK (required_classes > 0),
    scheduled_count INTEGER DEFAULT 0 CHECK (scheduled_count >= 0),
    completed_count INTEGER DEFAULT 0 CHECK (completed_count >= 0),
    status TEXT CHECK (status IN ('fulfilled', 'scheduled', 'unscheduled', 'skipped')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (course_id) REFERENCES courses(course_id) ON DELETE CASCADE,
    CHECK (period_end > period_start),
    CHECK (completed_count <= scheduled_count),
    CHECK (scheduled_count <= required_classes)
);

-- 10. TrackerClass Junction Table
CREATE TABLE tracker_classes (
    tracking_id UUID NOT NULL,
    class_id UUID NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('scheduled', 'completed')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (tracking_id, class_id),
    FOREIGN KEY (tracking_id) REFERENCES trackers(tracking_id) ON DELETE CASCADE,
    FOREIGN KEY (class_id) REFERENCES classes(class_id) ON DELETE CASCADE
);