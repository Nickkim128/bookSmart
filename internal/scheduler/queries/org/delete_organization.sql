-- Delete organization (will cascade to users and courses)
delete from organizations
where organization_id = $1;
