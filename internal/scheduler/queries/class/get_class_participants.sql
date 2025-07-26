select
	user_id,
	role
from class_participants
where class_id = $1
order by role, user_id;
