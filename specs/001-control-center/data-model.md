# Data Model: Control Center

## Job

- id, source, source_id, url, title, company, location?, is_remote, is_relocation, salary_min?, salary_max?, description, status, created_at
- status ∈ discovered | scored | queued | approved | applied | rejected | interview

## Match (0..1 per Job)

- job_id (unique), score (0–1), track (fullstack|blockchain|fde), matched_skills (JSON array string), missing_skills, score_reasons, scored_at

## Application (0..1 per Job)

- job_id (unique), resume_path (`resumes/{track}.pdf`), cover_letter, custom_qa?, submission_method, submitted_at?, auto_applied, submission_status, response_status, notes?

## JobListItem (API view)

- Job fields + optional score, track from Match join

## JobDetail (API view)

- job + match? + application?

## Stats

- total_jobs, total_applications, applied, interviews, by_status map
- daily_limit guidance: env `DAILY_APPLICATION_LIMIT` (default 8) exposed on stats or frontend config

## Validation rules

- resume track must map to one of three paths
- discard → status rejected
- approve without successful submit → must not claim applied
