-- +goose Up
ALTER TABLE job_attempts
    ADD CONSTRAINT job_attempts_job_id_attempt_num_unique UNIQUE (job_id, attempt_num);

-- +goose Down
ALTER TABLE job_attempts
    DROP CONSTRAINT job_attempts_job_id_attempt_num_unique;
