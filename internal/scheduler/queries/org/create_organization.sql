-- Create a new organization
insert into organizations (organization_id, name)
values ($1, $2)
returning organization_id, name, created_at, updated_at;
