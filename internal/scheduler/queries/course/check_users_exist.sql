select
	user_id,
	role,
	org_id
from users
where user_id = ANY($1) and org_id = $2;
