select
	availability_id,
	user_id,
	start_time,
	end_time
from (
	select
		availability_id,
		user_id,
		start_time,
		end_time,
		row_number() over (
			partition by user_id, start_time
			order by created_at desc
		) as rn
	from availability
	where user_id = $1
) as ranked
where rn = 1
order by start_time;
