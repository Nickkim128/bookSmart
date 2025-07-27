insert into user_courses (user_id, course_id, enrolled_at)
values ($1, $2, $3)
on conflict (user_id, course_id) do nothing;
