package api

import (
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRouter(h *Handler, auth *AuthHandler) *gin.Engine {
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
	config.AllowCredentials = true
	config.ExposeHeaders = []string{"Authorization"}
	r.Use(cors.New(config))

	v1 := r.Group("")
	{
		// Scams
		scams := v1.Group("/scams")
		{
			scams.POST("", OptionalJWTMiddleware(), h.CreateScam)
			scams.GET("", h.SearchScams)
			scams.POST("/check-duplicates", h.CheckDuplicates)
			scams.GET("/:id", h.GetScam)
			scams.GET("/:id/similar", h.FindSimilarScams)
			scams.POST("/:id/report", h.ReportScam)
			scams.POST("/:id/experienced", OptionalJWTMiddleware(), h.ExperiencedScam)
			scams.GET("/:id/comments", h.GetComments)
			scams.POST("/:id/comments", h.CreateComment)
			scams.POST("/:id/contact-methods", JWTMiddleware(), h.AddContactMethod)
			scams.POST("/:id/transfer-methods", JWTMiddleware(), h.AddTransferMethod)
			scams.POST("/:id/locations", JWTMiddleware(), h.AddLocation)
			scams.POST("/:id/keywords", JWTMiddleware(), h.AddKeyword)
		}

		// Me
		v1.GET("/me/activity", JWTMiddleware(), h.GetMyActivity)

		// Auth
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", auth.Register)
			authGroup.POST("/login", auth.Login)
			authGroup.POST("/logout", auth.Logout)
			authGroup.GET("/me", JWTMiddleware(), auth.Me)
			authGroup.GET("/google", auth.GoogleLogin)
			authGroup.GET("/google/callback", auth.GoogleCallback)
		}

		// Identifier lookup (SEO)
		v1.GET("/lookup/:identifier", h.LookupScam)

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
