select
	course_id,
	course_name,
	course_description
from courses
where org_id = $1;
