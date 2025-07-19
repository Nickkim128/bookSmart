-- Migration: 001_create_tables.sql
-- Description: Create initial database schema for Scheduler system
-- Compatible with: PostgreSQL/Neon

-- Enable UUID extension for generating UUIDs
create extension if not exists "uuid-ossp";

-- 1. Organization Table
create table organizations (
	organization_id UUID primary key default uuid_generate_v4(),
	name TEXT not null,
	created_at TIMESTAMPTZ default now(),
	updated_at TIMESTAMPTZ default now()
);

-- 2. User Table
create table users (
	user_id UUID primary key default uuid_generate_v4(),
	org_id UUID not null,
	role TEXT not null check (role in ('admin', 'student', 'tutor')),
	first_name TEXT not null,
	last_name TEXT not null,
	phone_number TEXT,
	email TEXT,
	created_at TIMESTAMPTZ default now(),
	updated_at TIMESTAMPTZ default now(),
	foreign key (org_id) references organizations (organization_id) on delete cascade
);

-- 3. Course Table
create table courses (
	course_id UUID primary key default uuid_generate_v4(),
	org_id UUID not null,
	course_name TEXT not null,
	course_description TEXT,
	start_at TIMESTAMPTZ,
	end_at TIMESTAMPTZ,
	interval TEXT check (interval in ('week', 'bi-weekly', 'month')),
	frequency INTEGER,
	created_at TIMESTAMPTZ default now(),
	updated_at TIMESTAMPTZ default now(),
	foreign key (org_id) references organizations (organization_id) on delete cascade,
	check (end_at > start_at or (end_at is NULL or start_at is NULL))
);

-- 4. UserCourse Junction Table
create table user_courses (
	user_id UUID not null,
	course_id UUID not null,
	enrolled_at TIMESTAMPTZ default now(),
	status TEXT default 'active' check (status in ('active', 'completed', 'dropped')),
	primary key (user_id, course_id),
	foreign key (user_id) references users (user_id) on delete cascade,
	foreign key (course_id) references courses (course_id) on delete cascade
);

-- 5. Availability Table
create table availability (
	availability_id UUID primary key default uuid_generate_v4(),
	org_id UUID not null,
	user_id UUID not null,
	role TEXT not null check (role in ('admin', 'student', 'tutor')),
	start_time TIMESTAMPTZ not null,
	end_time TIMESTAMPTZ not null,
	matched BOOLEAN not null default FALSE,
	created_at TIMESTAMPTZ default now(),
	updated_at TIMESTAMPTZ default now(),
	foreign key (org_id) references organizations (organization_id) on delete cascade,
	foreign key (user_id) references users (user_id) on delete cascade,
	check (end_time > start_time),
	check (extract(epoch from (end_time - start_time)) >= 900) -- Minimum 15 minutes
);

-- 6. Class Table
create table classes (
	class_id UUID primary key default uuid_generate_v4(),
	course_id UUID,
	org_id UUID not null,
	start_time TIMESTAMPTZ not null,
	duration INTEGER not null check (duration > 0), -- Duration in minutes
	created_at TIMESTAMPTZ default now(),
	updated_at TIMESTAMPTZ default now(),
	foreign key (course_id) references courses (course_id) on delete set null,
	foreign key (org_id) references organizations (organization_id) on delete cascade
);

-- 7. ClassParticipant Table
create table class_participants (
	class_id UUID not null,
	user_id UUID not null,
	role TEXT not null check (role in ('student', 'teacher')),
	created_at TIMESTAMPTZ default now(),
	primary key (class_id, user_id),
	foreign key (class_id) references classes (class_id) on delete cascade,
	foreign key (user_id) references users (user_id) on delete cascade
);

-- 8. ClassAttendance Table
create table class_attendance (
	class_id UUID not null,
	user_id UUID not null,
	role TEXT not null check (role in ('student', 'teacher')),
	attended BOOLEAN default NULL, -- NULL = not yet determined
	notes TEXT,
	recorded_at TIMESTAMPTZ default now(),
	primary key (class_id, user_id),
	foreign key (class_id) references classes (class_id) on delete cascade,
	foreign key (user_id) references users (user_id) on delete cascade
);

-- 9. Tracker Table
create table trackers (
	tracking_id UUID primary key default uuid_generate_v4(),
	course_id UUID not null,
	period_start TIMESTAMPTZ not null,
	period_end TIMESTAMPTZ not null,
	required_classes INTEGER not null check (required_classes > 0),
	scheduled_count INTEGER default 0 check (scheduled_count >= 0),
	completed_count INTEGER default 0 check (completed_count >= 0),
	status TEXT check (status in ('fulfilled', 'scheduled', 'unscheduled', 'skipped')),
	created_at TIMESTAMPTZ default now(),
	updated_at TIMESTAMPTZ default now(),
	foreign key (course_id) references courses (course_id) on delete cascade,
	check (period_end > period_start),
	check (completed_count <= scheduled_count),
	check (scheduled_count <= required_classes)
);

-- 10. TrackerClass Junction Table
create table tracker_classes (
	tracking_id UUID not null,
	class_id UUID not null,
	status TEXT not null check (status in ('scheduled', 'completed')),
	created_at TIMESTAMPTZ default now(),
	primary key (tracking_id, class_id),
	foreign key (tracking_id) references trackers (tracking_id) on delete cascade,
	foreign key (class_id) references classes (class_id) on delete cascade
);
