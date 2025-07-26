select exists(
	select 1
	from courses as c
	left join course_students as cs on c.course_id = cs.course_id
	where
		c.course_id = $1
		and c.org_id = $3
		and (
			cs.user_id = $2
			or c.course_id in (
				select cs2.course_id
				from course_students as cs2
				where cs2.user_id = $2
			)
		)
);
