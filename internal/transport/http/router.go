package httptransport

import (
	"net/http"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/usecase"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	// Schedules a job to be executed later, calls UseCase/Service
	r.POST("/jobs", func(ctx *gin.Context) {
		_, err := usecase.CreateJob(usecase.CreateJobInput{})
		if err != nil {
			// is this 400 or 500 error?
		}

		ctx.JSON(http.StatusOK, gin.H{
			"message": "Success!",
		})
	})

	return r
}
