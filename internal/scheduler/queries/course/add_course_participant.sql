insert into user_courses (user_id, course_id, role, created_at)
values ($1, $2, $3, $4)
on conflict (user_id, course_id) do nothing;
