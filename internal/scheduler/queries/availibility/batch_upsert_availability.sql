insert into availability (availability_id, org_id, user_id, role, start_time, end_time, matched, created_at, updated_at)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
on conflict (user_id, start_time, end_time)
do update set
	org_id = excluded.org_id,
	role = excluded.role,
	matched = excluded.matched,
	updated_at = excluded.updated_at;
