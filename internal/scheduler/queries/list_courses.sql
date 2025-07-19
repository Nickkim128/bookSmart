SELECT
	course_id,
	course_name,
	course_description
FROM courses
WHERE org_id = $1;
