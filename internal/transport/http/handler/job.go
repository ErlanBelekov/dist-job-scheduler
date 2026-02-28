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
		h.logger.Error("create job", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusCreated, job)
}

// Get a Job by ID
func (h *JobHandler) GetByID(ctx *gin.Context) {
	jobID := ctx.Param("id")

	job, err := h.jobUsecase.GetByID(ctx.Request.Context(), jobID)
	if err != nil {
		if errors.Is(err, domain.ErrJobNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		h.logger.Error("get job by id", "job_id", jobID, "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusOK, job)
}
