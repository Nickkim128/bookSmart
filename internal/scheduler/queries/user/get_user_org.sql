-- Get user's organization and role for validation
select
	org_id,
	role
from users
where firebase_uid = $1 and status = 'active';
