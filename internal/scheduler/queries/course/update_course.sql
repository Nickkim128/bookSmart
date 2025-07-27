update courses
set
	course_name = coalesce($2, course_name),
	course_description = coalesce($3, course_description),
	start_at = coalesce($4, start_at),
	end_at = coalesce($5, end_at),
	interval = coalesce($6, interval),
	frequency = coalesce($7, frequency),
	updated_at = $8
where course_id = $1;
