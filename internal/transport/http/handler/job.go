package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/usecase"
	"github.com/gin-gonic/gin"
)

type JobHandler struct {
	jobUsecase *usecase.JobUsecase
	logger     *slog.Logger
}

func NewJobHandler(jobUsecase *usecase.JobUsecase, logger *slog.Logger) *JobHandler {
	return &JobHandler{jobUsecase: jobUsecase, logger: logger.With("component", "job_handler")}
}

type createJobRequest struct {
	IdempotencyKey string            `json:"idempotency_key" binding:"required"`
	URL            string            `json:"url"             binding:"required,url"`
	Method         string            `json:"method"          binding:"required,oneof=GET POST PUT PATCH DELETE"`
	Headers        map[string]string `json:"headers"`
	Body           *string           `json:"body"`
	TimeoutSeconds int               `json:"timeout_seconds"`
	ScheduledAt    time.Time         `json:"scheduled_at"    binding:"required"`
	MaxRetries     int               `json:"max_retries"`
	Backoff        domain.Backoff    `json:"backoff"         binding:"omitempty,oneof=exponential linear"`
}

type createJobResponse struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type getJobResponse struct {
	ID          string         `json:"id"`
	Status      domain.Status  `json:"status"`
	ScheduledAt time.Time      `json:"scheduled_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	LastError   *string        `json:"last_error,omitempty"`
}

func (h *JobHandler) Create(ctx *gin.Context) {
	var req createJobRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job, err := h.jobUsecase.CreateJob(ctx.Request.Context(), usecase.CreateJobInput{
		IdempotencyKey: req.IdempotencyKey,
		URL:            req.URL,
		Method:         req.Method,
		Headers:        req.Headers,
		Body:           req.Body,
		TimeoutSeconds: req.TimeoutSeconds,
		ScheduledAt:    req.ScheduledAt,
		MaxRetries:     req.MaxRetries,
		Backoff:        req.Backoff,
	})
	if err != nil {
		if errors.Is(err, domain.ErrDuplicateJob) {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": errDuplicateJob})
			return
		}
		h.logger.Error("create job", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": errInternalServer})
		return
	}

	ctx.JSON(http.StatusCreated, createJobResponse{
		ID:        job.ID,
		CreatedAt: job.CreatedAt,
	})
}

func (h *JobHandler) GetByID(ctx *gin.Context) {
	jobID := ctx.Param("id")

	job, err := h.jobUsecase.GetByID(ctx.Request.Context(), jobID)
	if err != nil {
		if errors.Is(err, domain.ErrJobNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": errJobNotFound})
			return
		}
		h.logger.Error("get job by id", "job_id", jobID, "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": errInternalServer})
		return
	}

	ctx.JSON(http.StatusOK, getJobResponse{
		ID:          job.ID,
		Status:      job.Status,
		ScheduledAt: job.ScheduledAt,
		CreatedAt:   job.CreatedAt,
		UpdatedAt:   job.UpdatedAt,
		CompletedAt: job.CompletedAt,
		LastError:   job.LastError,
	})
}
