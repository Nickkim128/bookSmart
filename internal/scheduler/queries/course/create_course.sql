insert into courses (
	course_id,
	org_id,
	course_name,
	course_description,
	start_at,
	end_at,
	interval,
	frequency,
	created_at,
	updated_at
) values (
	$1, $2, $3, $4, $5, $6, $7, $8,
	NOW(), NOW()
) returning
	course_id,
	org_id,
	course_name,
	course_description,
	start_at,
	end_at,
	interval,
	frequency,
	created_at,
	updated_at;
