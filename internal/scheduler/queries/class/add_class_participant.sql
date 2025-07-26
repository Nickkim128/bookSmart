insert into class_participants (class_id, user_id, role, created_at)
values ($1, $2, $3, NOW())
on conflict (class_id, user_id) do nothing;
