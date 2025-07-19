-- Migration: 002_create_indexes.sql
-- Description: Create performance indexes for Scheduler system
-- Compatible with: PostgreSQL/Neon

-- Organization indexes
create index idx_organizations_name on organizations (name);

-- User indexes
create index idx_users_org_id on users (org_id);
create index idx_users_role on users (role);
create index idx_users_email on users (email);
create index idx_users_org_role on users (org_id, role);

-- Course indexes
create index idx_courses_org_id on courses (org_id);
create index idx_courses_name on courses (course_name);
create index idx_courses_time_range on courses (start_at, end_at);
create index idx_courses_interval on courses (interval);

-- UserCourse indexes
create index idx_user_courses_user_id on user_courses (user_id);
create index idx_user_courses_course_id on user_courses (course_id);
create index idx_user_courses_status on user_courses (status);

-- Availability indexes (critical for calendar queries)
create index idx_availability_user_id on availability (user_id);
create index idx_availability_org_id on availability (org_id);
create index idx_availability_time_range on availability (start_time, end_time);
create index idx_availability_user_time on availability (user_id, start_time, end_time);
create index idx_availability_org_time on availability (org_id, start_time, end_time);
create index idx_availability_role on availability (role);
create index idx_availability_matched on availability (matched);
create index idx_availability_unmatched_time on availability (start_time, end_time) where matched = FALSE;

-- Class indexes (critical for scheduling queries)
create index idx_classes_course_id on classes (course_id);
create index idx_classes_org_id on classes (org_id);
create index idx_classes_start_time on classes (start_time);
create index idx_classes_time_range on classes (start_time, (start_time + INTERVAL '1 minute' * duration));
create index idx_classes_org_time on classes (org_id, start_time);

-- ClassParticipant indexes
create index idx_class_participants_user_id on class_participants (user_id);
create index idx_class_participants_class_id on class_participants (class_id);
create index idx_class_participants_role on class_participants (role);
create index idx_class_participants_user_role on class_participants (user_id, role);

-- ClassAttendance indexes
create index idx_class_attendance_user_id on class_attendance (user_id);
create index idx_class_attendance_class_id on class_attendance (class_id);
create index idx_class_attendance_attended on class_attendance (attended);
create index idx_class_attendance_user_attended on class_attendance (user_id, attended);

-- Tracker indexes
create index idx_trackers_course_id on trackers (course_id);
create index idx_trackers_period on trackers (period_start, period_end);
create index idx_trackers_status on trackers (status);
create index idx_trackers_course_period on trackers (course_id, period_start, period_end);

-- TrackerClass indexes
create index idx_tracker_classes_tracking_id on tracker_classes (tracking_id);
create index idx_tracker_classes_class_id on tracker_classes (class_id);
create index idx_tracker_classes_status on tracker_classes (status);

-- Composite indexes for common query patterns
create index idx_users_org_role_name on users (org_id, role, first_name, last_name);
create index idx_availability_search on availability (org_id, role, start_time, end_time, matched);
create index idx_classes_search on classes (org_id, start_time, duration);
create index idx_class_participants_search on class_participants (user_id, role, class_id);

-- Partial indexes for better performance on filtered queries
create index idx_active_user_courses on user_courses (user_id, course_id) where status = 'active';
create index idx_upcoming_classes on classes (org_id, start_time) where start_time > NOW();
create index idx_pending_attendance on class_attendance (class_id, user_id) where attended is NULL;
