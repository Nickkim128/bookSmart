-- Check if organization exists
select COUNT(*)
from organizations
where organization_id = $1;
