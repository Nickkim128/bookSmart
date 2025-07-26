select
	class_id,
	course_id,
	start_time,
	duration
from classes
where course_id = $1
order by start_time;
