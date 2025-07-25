select course_id
from user_courses
where user_id = $1 and status = 'active';
