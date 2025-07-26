delete from course_students
where user_id = $1 and course_id = $2;
