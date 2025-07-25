select
	availability_id,
	user_id,
	start_time,
	end_time
from availability
where user_id = $1
order by start_time;
