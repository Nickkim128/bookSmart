# Database Setup for Scheduler System

This document explains how to set up and deploy the PostgreSQL database schema for the Scheduler system, specifically optimized for Neon database deployment.

## Quick Start with Neon

1. **Create a Neon project** at [neon.tech](https://neon.tech)
2. **Copy your DATABASE_URL** from the Neon dashboard
3. **Set up environment variables**:
   ```bash
   cp .env.example .env
   # Edit .env and set your DATABASE_URL
   ```
4. **Run migrations**:
   ```bash
   go run cmd/migrate/main.go -neon
   ```

## Database Schema Overview

The database consists of 10 tables supporting a multi-tenant scheduling system:

### Core Tables

- **organizations** - Multi-tenant organization management
- **users** - Students, tutors, and administrators
- **courses** - Recurring course definitions
- **classes** - Individual class sessions

### Relationship Tables

- **user_courses** - Many-to-many user-course enrollments
- **class_participants** - Class participants (students/teachers)
- **class_attendance** - Attendance tracking
- **availability** - User availability in 15-minute blocks

### Progress Tracking

- **trackers** - Course progress monitoring
- **tracker_classes** - Links classes to tracking periods

## Files Structure

```
/migrations/
├── 001_create_tables.sql    # Core schema with foreign keys and constraints
├── 002_create_indexes.sql   # Performance indexes for queries
└── 003_sample_data.sql      # Sample data for testing

/database/
└── config.go               # Database configuration and connection

/cmd/migrate/
└── main.go                 # Migration runner utility

.env.example               # Environment configuration template
```

## Environment Configuration

### Option 1: DATABASE_URL (Recommended for Neon)

```bash
DATABASE_URL=postgresql://username:password@ep-example.us-east-1.aws.neon.tech/dbname?sslmode=require
```

### Option 2: Individual Environment Variables

```bash
DB_HOST=your-neon-host
DB_PORT=5432
DB_USER=your-username
DB_PASSWORD=your-password
DB_NAME=your-database
DB_SSL_MODE=require
```

## Running Migrations

### Local Development

```bash
# Install PostgreSQL driver dependency
go mod tidy

# Run all migrations
go run cmd/migrate/main.go

# Specify custom migrations directory
go run cmd/migrate/main.go -migrations-dir ./custom/path
```

### Neon Deployment

```bash
# Use Neon configuration
go run cmd/migrate/main.go -neon

# Or with custom directory
go run cmd/migrate/main.go -neon -migrations-dir ./migrations
```

## Key Features

### 1. Multi-Tenant Architecture

- All tables include `org_id` for organization isolation
- Foreign key constraints maintain data integrity
- Proper CASCADE/SET NULL relationships

### 2. Performance Optimizations

- Comprehensive indexing strategy for calendar queries
- Partial indexes for filtered queries (active courses, upcoming classes)
- Composite indexes for common query patterns

### 3. Data Validation

- CHECK constraints for role validation
- Time range validation (end > start)
- Connection pool optimization for Neon

### 4. PostgreSQL/Neon Specific Features

- UUID primary keys with `uuid_generate_v4()`
- TIMESTAMPTZ for proper timezone handling
- SSL required for Neon connections
- Connection pool limits optimized for Neon

## Sample Data

The system includes realistic sample data:

- 2 organizations (Bright Minds Tutoring, Excellence Academy)
- 9 users (admins, tutors, students)
- 4 courses (Algebra I, Geometry, SAT Prep, Calculus AB)
- Availability patterns and scheduled classes
- Progress tracking examples

## Common Queries

### Find Available Time Slots

```sql
SELECT a1.user_id as tutor_id, a2.user_id as student_id,
       GREATEST(a1.start_time, a2.start_time) as start_time,
       LEAST(a1.end_time, a2.end_time) as end_time
FROM availability a1
JOIN availability a2 ON a1.org_id = a2.org_id
WHERE a1.role = 'tutor' AND a2.role = 'student'
  AND a1.matched = FALSE AND a2.matched = FALSE
  AND GREATEST(a1.start_time, a2.start_time) < LEAST(a1.end_time, a2.end_time);
```

### Get User's Upcoming Classes

```sql
SELECT c.class_id, c.start_time, c.duration,
       co.course_name, cp.role
FROM classes c
JOIN class_participants cp ON c.class_id = cp.class_id
LEFT JOIN courses co ON c.course_id = co.course_id
WHERE cp.user_id = $1 AND c.start_time > NOW()
ORDER BY c.start_time;
```

## Production Considerations

1. **Connection Pooling**: Pre-configured for Neon's connection limits
2. **SSL Requirements**: Enforced for Neon deployments
3. **Migration Tracking**: Built-in migration versioning system
4. **Error Handling**: Comprehensive error handling in connection setup
5. **Performance**: Optimized indexes for calendar and scheduling queries

## Troubleshooting

### Connection Issues

- Verify DATABASE_URL format includes `?sslmode=require` for Neon
- Check Neon project status and connection limits
- Ensure proper network access to Neon endpoint

### Migration Failures

- Check migration file syntax (PostgreSQL specific)
- Verify proper file permissions
- Review migration logs for specific error messages

### Performance Issues

- Monitor connection pool usage
- Review query execution plans
- Consider additional indexes for specific query patterns
