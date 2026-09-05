package rest

import (
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/adapters/rest/handler"
	"github.com/danila-kuryakin/banking_mini_cores/services/api-gateway/internal/domain"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Router struct {
	handler *handler.Handler
}

func NewRouter(handler *handler.Handler) *Router {
	return &Router{
		handler: handler,
	}
}

func (r *Router) Init(g *gin.Engine) {
	api := g.Group("/api/v1")

	r.initAuthRoutes(api)
	r.initCustomerRoutes(api)

	g.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func (r *Router) initAuthRoutes(api *gin.RouterGroup) {
	auth := api.Group("/auth")
	{
		auth.POST("/register", r.handler.Auth.Register)
		auth.POST("/login", r.handler.Auth.Login)

		auth.POST("/refresh", r.handler.Auth.RefreshToken)
		auth.POST("/logout", r.handler.Auth.Logout)

		auth.POST("/validate", r.handler.Auth.ValidateToken)
		auth.POST("/officers", r.handler.Middleware.RequireAuth(), r.handler.Middleware.RequireRole(domain.ROLE_ADMIN), r.handler.Auth.CreateOfficers)
	}
}

func (r *Router) initCustomerRoutes(api *gin.RouterGroup) {
	customers := api.Group("/customer", r.handler.Middleware.RequireAuth())
	{
		customers.GET("/:customer_id", r.handler.Customer.GetCustomer)
		customers.POST("/:customer_id", r.handler.Customer.UpdateProfile)

		customers.GET("/:customer_id/status", r.handler.Customer.GetCustomerStatus)
		customers.GET("", r.handler.Middleware.RequireRole(domain.ROLE_OFFICER, domain.ROLE_ADMIN), r.handler.Customer.ListCustomers)
	}
}
