select
	course_id,
	course_name,
	course_description
from courses
where course_id = $1;
