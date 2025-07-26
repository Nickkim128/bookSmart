insert into course_students (user_id, course_id, created_at)
values ($1, $2, NOW())
on conflict (user_id, course_id) do nothing;
