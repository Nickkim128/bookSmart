-- Migration: 002_create_indexes.sql
-- Description: Create performance indexes for Scheduler system
-- Compatible with: PostgreSQL/Neon

-- Organization indexes
CREATE INDEX idx_organizations_name ON organizations(name);

-- User indexes
CREATE INDEX idx_users_org_id ON users(org_id);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_org_role ON users(org_id, role);

-- Course indexes
CREATE INDEX idx_courses_org_id ON courses(org_id);
CREATE INDEX idx_courses_name ON courses(course_name);
CREATE INDEX idx_courses_time_range ON courses(start_at, end_at);
CREATE INDEX idx_courses_interval ON courses(interval);

-- UserCourse indexes
CREATE INDEX idx_user_courses_user_id ON user_courses(user_id);
CREATE INDEX idx_user_courses_course_id ON user_courses(course_id);
CREATE INDEX idx_user_courses_status ON user_courses(status);

-- Availability indexes (critical for calendar queries)
CREATE INDEX idx_availability_user_id ON availability(user_id);
CREATE INDEX idx_availability_org_id ON availability(org_id);
CREATE INDEX idx_availability_time_range ON availability(start_time, end_time);
CREATE INDEX idx_availability_user_time ON availability(user_id, start_time, end_time);
CREATE INDEX idx_availability_org_time ON availability(org_id, start_time, end_time);
CREATE INDEX idx_availability_role ON availability(role);
CREATE INDEX idx_availability_matched ON availability(matched);
CREATE INDEX idx_availability_unmatched_time ON availability(start_time, end_time) WHERE matched = FALSE;

-- Class indexes (critical for scheduling queries)
CREATE INDEX idx_classes_course_id ON classes(course_id);
CREATE INDEX idx_classes_org_id ON classes(org_id);
CREATE INDEX idx_classes_start_time ON classes(start_time);
-- CREATE INDEX idx_classes_time_range ON classes(start_time, (start_time + INTERVAL '1 minute' * duration));
-- Note: Computed end time index removed due to PostgreSQL immutability requirements
CREATE INDEX idx_classes_start_duration ON classes(start_time, duration);
CREATE INDEX idx_classes_org_time ON classes(org_id, start_time);

-- ClassParticipant indexes
CREATE INDEX idx_class_participants_user_id ON class_participants(user_id);
CREATE INDEX idx_class_participants_class_id ON class_participants(class_id);
CREATE INDEX idx_class_participants_role ON class_participants(role);
CREATE INDEX idx_class_participants_user_role ON class_participants(user_id, role);

-- ClassAttendance indexes
CREATE INDEX idx_class_attendance_user_id ON class_attendance(user_id);
CREATE INDEX idx_class_attendance_class_id ON class_attendance(class_id);
CREATE INDEX idx_class_attendance_attended ON class_attendance(attended);
CREATE INDEX idx_class_attendance_user_attended ON class_attendance(user_id, attended);

-- Tracker indexes
CREATE INDEX idx_trackers_course_id ON trackers(course_id);
CREATE INDEX idx_trackers_period ON trackers(period_start, period_end);
CREATE INDEX idx_trackers_status ON trackers(status);
CREATE INDEX idx_trackers_course_period ON trackers(course_id, period_start, period_end);

-- TrackerClass indexes
CREATE INDEX idx_tracker_classes_tracking_id ON tracker_classes(tracking_id);
CREATE INDEX idx_tracker_classes_class_id ON tracker_classes(class_id);
CREATE INDEX idx_tracker_classes_status ON tracker_classes(status);

-- Composite indexes for common query patterns
CREATE INDEX idx_users_org_role_name ON users(org_id, role, first_name, last_name);
CREATE INDEX idx_availability_search ON availability(org_id, role, start_time, end_time, matched);
CREATE INDEX idx_classes_search ON classes(org_id, start_time, duration);
CREATE INDEX idx_class_participants_search ON class_participants(user_id, role, class_id);

-- Partial indexes for better performance on filtered queries
CREATE INDEX idx_active_user_courses ON user_courses(user_id, course_id) WHERE status = 'active';
-- CREATE INDEX idx_upcoming_classes ON classes(org_id, start_time) WHERE start_time > NOW();
-- Note: NOW() function index removed due to PostgreSQL immutability requirements
-- Query performance can be achieved with regular index on (org_id, start_time)
CREATE INDEX idx_pending_attendance ON class_attendance(class_id, user_id) WHERE attended IS NULL;