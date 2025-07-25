select
	user_id,
	org_id,
	first_name,
	last_name,
	phone_number,
	role,
	email
from users
where user_id = $1;
