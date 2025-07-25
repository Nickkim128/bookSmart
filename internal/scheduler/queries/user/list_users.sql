select
	user_id,
	org_id,
	first_name,
	last_name,
	phone_number,
	role,
	email
from users
where org_id = $1;
