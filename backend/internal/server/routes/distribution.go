package routes

import (
	"fmt"

	"github.com/WilliamWang1721/LightBridge/internal/handler"
	adminhandler "github.com/WilliamWang1721/LightBridge/internal/handler/admin"
	"github.com/WilliamWang1721/LightBridge/internal/repository"
	"github.com/WilliamWang1721/LightBridge/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterDistributionRoutes registers the user inbox and administrator
// distribution console using the application's already initialized runtime
// dependencies. It intentionally stays outside the generated Wire graph.
func RegisterDistributionRoutes(
	v1 *gin.RouterGroup,
	jwtAuth middleware.JWTAuthMiddleware,
	adminAuth middleware.AdminAuthMiddleware,
) {
	distributionService, err := repository.BuildRuntimeDistributionService()
	if err != nil {
		panic(fmt.Sprintf("initialize distribution routes: %v", err))
	}

	userHandler := handler.NewDistributionHandler(distributionService)
	userRoutes := v1.Group("/distributions")
	userRoutes.Use(gin.HandlerFunc(jwtAuth))
	{
		userRoutes.GET("", userHandler.List)
		userRoutes.GET("/:id", userHandler.Get)
		userRoutes.POST("/:id/read", userHandler.MarkRead)
		userRoutes.POST("/:id/accept", userHandler.Accept)
		userRoutes.POST("/:id/reject", userHandler.Reject)
		userRoutes.GET("/:id/download", userHandler.Download)
	}

	adminHandler := adminhandler.NewDistributionHandler(distributionService)
	adminRoutes := v1.Group("/admin/distributions")
	adminRoutes.Use(gin.HandlerFunc(adminAuth))
	{
		adminRoutes.GET("", adminHandler.List)
		adminRoutes.POST("", adminHandler.Create)
		adminRoutes.POST("/batch", adminHandler.BatchCreate)
		adminRoutes.POST("/audience-preview", adminHandler.PreviewAudience)
		adminRoutes.GET("/:id", adminHandler.Get)
		adminRoutes.GET("/:id/download", adminHandler.Download)
		adminRoutes.DELETE("/:id", adminHandler.Delete)
	}
}
