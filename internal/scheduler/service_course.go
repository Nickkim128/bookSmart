package scheduler

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"scheduler-api/internal/auth"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CourseService interface {
	CreateCourse(*gin.Context)
	GetCourse(*gin.Context, string)
	ListCourses(*gin.Context)
	UpdateCourse(*gin.Context, string)
}

var _ CourseService = (*Service)(nil)

func (s *Service) CreateCourse(c *gin.Context) {
	currentUser, err := auth.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	if currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Only admin can create courses",
		})
		return
	}

	createCourseRequest := Course{}
	if err := c.ShouldBindJSON(&createCourseRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var (
		// Hard coded org id here for now.
		orgID = "00000000-0000-0000-0000-000000000001"
		now   = time.Now()
	)

	err = createCourse(c.Request.Context(), s.pgxPool, createCourseRequest, orgID, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Failed to create Course": err.Error()})
		return
	}

	courseParticipantsError := addCourseParticipants(c.Request.Context(), s.pgxPool, createCourseRequest, now)

	if courseParticipantsError != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Failed to add course participants": courseParticipantsError.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Course created successfully"})
}

func (s *Service) GetCourse(c *gin.Context, courseID string) {
	course, err := getCourse(c.Request.Context(), s.pgxPool, courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	students, tutors, err := getCourseParticipants(c.Request.Context(), s.pgxPool, courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	course.Students = students
	course.Tutors = tutors
	c.JSON(http.StatusOK, course)
}

func (s *Service) ListCourses(c *gin.Context) {
	organizationID := "00000000-0000-0000-0000-000000000001"
	courses, err := listCourses(c.Request.Context(), s.pgxPool, organizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i := range courses {
		students, tutors, err := getCourseParticipants(c.Request.Context(), s.pgxPool, courses[i].CourseId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		courses[i].Students = students
		courses[i].Tutors = tutors
	}

	c.JSON(http.StatusOK, courses)
}

func (s *Service) UpdateCourse(c *gin.Context, courseID string) {
	currentUser, err := auth.GetCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	if currentUser.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "Only admin can create courses",
		})
		return
	}

	updateRequest := CourseUpdate{}
	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Debug logging
	fmt.Printf("Update request received for course %s:\n", courseID)
	if updateRequest.CourseName != nil {
		fmt.Printf("  Course Name: %s\n", *updateRequest.CourseName)
	}
	if updateRequest.StartAt != nil {
		fmt.Printf("  Start At: %v\n", *updateRequest.StartAt)
	}
	if updateRequest.EndAt != nil {
		fmt.Printf("  End At: %v\n", *updateRequest.EndAt)
	}
	if updateRequest.Frequency != nil {
		fmt.Printf("  Frequency: %d\n", *updateRequest.Frequency)
	}
	if updateRequest.Interval != nil {
		fmt.Printf("  Interval: %s\n", *updateRequest.Interval)
	}
	if updateRequest.Students != nil {
		fmt.Printf("  Students: %+v\n", updateRequest.Students)
	}
	if updateRequest.Teachers != nil {
		fmt.Printf("  Teachers: %+v\n", updateRequest.Teachers)
	}

	now := time.Now()
	err = updateCourse(c.Request.Context(), s.pgxPool, courseID, updateRequest, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Handle participant updates
	err = updateCourseParticipants(c.Request.Context(), s.pgxPool, courseID, updateRequest, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Failed to update course participants": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Course updated successfully"})
}

//go:embed queries/course/get_course.sql
var queryGetCourseSQL string

//go:embed queries/course/list_courses.sql
var queryListCoursesSQL string

//go:embed queries/course/get_course_users.sql
var queryGetCourseUsersSQL string

//go:embed queries/course/create_course.sql
var createCourseSql string

//go:embed queries/course/update_course.sql
var updateCourseSQL string

//go:embed queries/course/add_course_participant.sql
var addCourseParticipantSQL string

func listCourses(ctx context.Context, pgxPool *pgxpool.Pool, organizationID string) ([]Course, error) {
	courses := []Course{}
	return courses, pgxscan.Select(ctx, pgxPool, &courses, queryListCoursesSQL, organizationID)
}

func getCourseUsers(ctx context.Context, pgxPool *pgxpool.Pool, courseID string) ([]string, error) {
	users := []string{}
	return users, pgxscan.Select(ctx, pgxPool, &users, queryGetCourseUsersSQL, courseID)
}

func getCourseParticipants(ctx context.Context, pgxPool *pgxpool.Pool, courseID string) ([]string, []string, error) {
	// Get all user IDs for this course
	userIDs, err := getCourseUsers(ctx, pgxPool, courseID)
	if err != nil {
		return nil, nil, err
	}

	var students []string
	var tutors []string

	// For each user, check their role
	for _, userID := range userIDs {
		var role string
		err := pgxPool.QueryRow(ctx, "SELECT role FROM users WHERE user_id = $1", userID).Scan(&role)
		if err != nil {
			fmt.Printf("Error getting user role for %s: %v\n", userID, err)
			continue
		}

		switch role {
		case "student":
			students = append(students, userID)
		case "tutor", "admin":
			tutors = append(tutors, userID)
		}
	}

	return students, tutors, nil
}

func getCourse(ctx context.Context, pgxPool *pgxpool.Pool, courseID string) (Course, error) {
	course := Course{}
	return course, pgxscan.Get(ctx, pgxPool, &course, queryGetCourseSQL, courseID)
}

func createCourse(ctx context.Context, pgxPool *pgxpool.Pool, course Course, orgId string, now time.Time) error {
	_, err := pgxPool.Exec(ctx, createCourseSql, course.CourseId, orgId, course.CourseName, course.CourseDescription, course.StartAt, course.EndAt, course.Interval, course.Frequency, now)
	return err
}

func updateCourse(ctx context.Context, pgxPool *pgxpool.Pool, courseID string, update CourseUpdate, now time.Time) error {
	// Handle nil pointers by converting to proper values for SQL
	var courseName interface{} = nil
	if update.CourseName != nil {
		courseName = *update.CourseName
	}

	var courseDescription interface{} = nil

	var startAt interface{} = nil
	if update.StartAt != nil {
		startAt = *update.StartAt
	}

	var endAt interface{} = nil
	if update.EndAt != nil {
		endAt = *update.EndAt
	}

	var interval interface{} = nil
	if update.Interval != nil && string(*update.Interval) != "" {
		interval = string(*update.Interval)
	}

	var frequency interface{} = nil
	if update.Frequency != nil {
		frequency = *update.Frequency
	}

	fmt.Printf("SQL Update values: courseID=%s, courseName=%v, courseDescription=%v, startAt=%v, endAt=%v, interval=%v, frequency=%v\n",
		courseID, courseName, courseDescription, startAt, endAt, interval, frequency)

	_, err := pgxPool.Exec(ctx, updateCourseSQL, courseID, courseName, courseDescription, startAt, endAt, interval, frequency, now)
	if err != nil {
		fmt.Printf("SQL Update error: %v\n", err)
	}
	return err
}

func addCourseParticipants(ctx context.Context, pgxPool *pgxpool.Pool, course Course, now time.Time) error {
	batch := &pgx.Batch{}

	for _, student := range course.Students {
		batch.Queue(addCourseParticipantSQL, student, course.CourseId, now)
	}

	for _, tutor := range course.Tutors {
		batch.Queue(addCourseParticipantSQL, tutor, course.CourseId, now)
	}

	batchResult := pgxPool.SendBatch(ctx, batch)
	defer func() {
		_ = batchResult.Close()
	}()

	for i := 0; i < len(course.Students)+len(course.Tutors); i++ {
		_, err := batchResult.Exec()
		if err != nil {
			return err
		}
	}

	return nil
}

func updateCourseParticipants(ctx context.Context, pgxPool *pgxpool.Pool, courseID string, update CourseUpdate, now time.Time) error {
	batch := &pgx.Batch{}

	// Handle student updates
	if update.Students != nil {
		// Add new students
		if update.Students.Add != nil {
			for _, studentID := range *update.Students.Add {
				batch.Queue(addCourseParticipantSQL, studentID, courseID, now)
			}
		}
		// Remove students
		if update.Students.Remove != nil {
			for _, studentID := range *update.Students.Remove {
				batch.Queue("DELETE FROM user_courses WHERE user_id = $1 AND course_id = $2", studentID, courseID)
			}
		}
	}

	// Handle teacher/tutor updates
	if update.Teachers != nil {
		// Add new teachers
		if update.Teachers.Add != nil {
			for _, teacherID := range *update.Teachers.Add {
				batch.Queue(addCourseParticipantSQL, teacherID, courseID, now)
			}
		}
		// Remove teachers
		if update.Teachers.Remove != nil {
			for _, teacherID := range *update.Teachers.Remove {
				batch.Queue("DELETE FROM user_courses WHERE user_id = $1 AND course_id = $2", teacherID, courseID)
			}
		}
	}

	// Execute the batch if there are any operations
	if batch.Len() > 0 {
		batchResult := pgxPool.SendBatch(ctx, batch)
		defer func() {
			_ = batchResult.Close()
		}()

		for i := 0; i < batch.Len(); i++ {
			_, err := batchResult.Exec()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
