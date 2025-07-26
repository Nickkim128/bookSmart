-- Get organization by ID
select
	organization_id,
	name,
	created_at,
	updated_at
from organizations
where organization_id = $1;
