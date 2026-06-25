package api

import (
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter(h *Handler) *gin.Engine {
	r := gin.Default()

	// CORS configuration
	allowedOrigin := os.Getenv("CORS_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:3000"
	}
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{allowedOrigin}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	v1 := r.Group("")
	{
		// Scams
		scams := v1.Group("/scams")
		{
			scams.POST("", h.CreateScam)
			scams.GET("", h.SearchScams)
			scams.POST("/check-duplicates", h.CheckDuplicates)
			scams.GET("/:id", h.GetScam)
			scams.GET("/:id/similar", h.FindSimilarScams)
			scams.POST("/:id/report", h.ReportScam)
		}

		// Scam types lookup
		v1.GET("/types", h.GetScamTypes)

		// Statistics
		v1.GET("/statistics", h.GetStatistics)

		// Health check
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "healthy",
			})
		})
	}

	return r
}
