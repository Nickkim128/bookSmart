-- Migration: 004_add_firebase_auth.sql
-- Description: Add Firebase authentication support to user management
-- Compatible with: PostgreSQL/Neon

-- Add Firebase UID column to users table
alter table users add column firebase_uid text;

-- Add unique constraint on firebase_uid to prevent duplicate mappings
alter table users add constraint unique_firebase_uid unique (firebase_uid);

-- Create index for efficient Firebase UID lookups
create index idx_users_firebase_uid on users (firebase_uid);

-- Add email verification status (Firebase handles this but we may want to track it)
alter table users add column email_verified boolean default false;

-- Add last login timestamp for user activity tracking
alter table users add column last_login_at timestamptz;

-- Add user status for account management
alter table users add column status text default 'active' check (status in ('active', 'inactive', 'suspended'));

-- Create index on email for efficient lookups during Firebase sync
create index idx_users_email_lookup on users (email) where email is not null;

-- Update existing users to have a default status
update users set status = 'active'
where status is null;

-- Add constraint to ensure Firebase UID is provided for new users
-- (We'll handle this in the application layer, but add a comment for documentation)
comment on column users.firebase_uid is 'Firebase UID for authentication - should be set for all active users';
comment on column users.email_verified is 'Email verification status from Firebase';
comment on column users.last_login_at is 'Timestamp of user last successful login';
comment on column users.status is 'User account status: active, inactive, or suspended';
