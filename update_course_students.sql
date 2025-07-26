-- Script: update_course_students.sql
-- Description: Add additional students to existing courses
-- This script adds new student enrollments to courses that already exist

-- Add more students to Algebra I course (course_id: 30000000-0000-0000-0000-000000000001)
-- Currently has Emma and James, adding Olivia
insert into user_courses (user_id, course_id, status) values
('10000000-0000-0000-0000-000000000006', '30000000-0000-0000-0000-000000000001', 'active'); -- Olivia to Algebra I

-- Add more students to Geometry course (course_id: 30000000-0000-0000-0000-000000000002)  
-- Currently has Olivia, adding Emma and James
insert into user_courses (user_id, course_id, status) values
('10000000-0000-0000-0000-000000000004', '30000000-0000-0000-0000-000000000002', 'active'), -- Emma to Geometry
('10000000-0000-0000-0000-000000000005', '30000000-0000-0000-0000-000000000002', 'active'); -- James to Geometry

-- Add more students to Calculus AB course (course_id: 30000000-0000-0000-0000-000000000004)
-- Currently has David from Excellence Academy, adding students from Bright Minds
insert into user_courses (user_id, course_id, status) values
('10000000-0000-0000-0000-000000000004', '30000000-0000-0000-0000-000000000004', 'active'), -- Emma to Calculus AB  
('10000000-0000-0000-0000-000000000005', '30000000-0000-0000-0000-000000000004', 'active'), -- James to Calculus AB
('10000000-0000-0000-0000-000000000006', '30000000-0000-0000-0000-000000000004', 'active'); -- Olivia to Calculus AB

-- Verify the updates by showing student enrollment counts per course
-- Uncomment the following lines to run verification queries:

/*
-- Show current enrollments by course
SELECT
    c.course_name,
    c.course_id,
    COUNT(uc.user_id) as total_students,
    STRING_AGG(u.first_name || ' ' || u.last_name, ', ') as student_names
FROM courses c
LEFT JOIN user_courses uc ON c.course_id = uc.course_id
LEFT JOIN users u ON uc.user_id = u.user_id AND u.role = 'student'
WHERE uc.status = 'active'
GROUP BY c.course_id, c.course_name
ORDER BY c.course_name;

-- Show students enrolled in multiple courses
SELECT
    u.first_name || ' ' || u.last_name as student_name,
    COUNT(uc.course_id) as course_count,
    STRING_AGG(c.course_name, ', ') as courses
FROM users u
JOIN user_courses uc ON u.user_id = uc.user_id
JOIN courses c ON uc.course_id = c.course_id
WHERE u.role = 'student' AND uc.status = 'active'
GROUP BY u.user_id, u.first_name, u.last_name
HAVING COUNT(uc.course_id) > 1
ORDER BY course_count DESC, student_name;
*/
