package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/usecase"
	"github.com/gin-gonic/gin"
)

type JobHandler struct {
	jobUsecase *usecase.JobUsecase
}

func NewJobHandler(jobUsecase *usecase.JobUsecase) *JobHandler {
	return &JobHandler{jobUsecase: jobUsecase}
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
		log.Printf("create job error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	ctx.JSON(http.StatusCreated, job)
}
