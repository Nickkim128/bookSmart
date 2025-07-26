select
	c.class_id,
	c.start_time,
	c.duration,
	c.course_id
from classes as c
inner join class_participants as cp on c.class_id = cp.class_id
where cp.user_id = $1
order by c.start_time;
