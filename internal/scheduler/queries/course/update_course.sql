update courses set
	course_name = COALESCE($2, course_name),
	course_description = COALESCE($3, course_description),
	start_at = COALESCE($4, start_at),
	end_at = COALESCE($5, end_at),
	interval = COALESCE($6, interval),
	frequency = COALESCE($7, frequency),
	updated_at = NOW()
where course_id = $1
returning
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
