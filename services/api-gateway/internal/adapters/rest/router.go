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
	r.initFileRoutes(api)

	g.GET(domain.SWAGGER_ROUTE, ginSwagger.WrapHandler(swaggerFiles.Handler))
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
		own := customers.Group("/:user_id",
			r.handler.Middleware.RequireSelfOrRole(domain.USER_ID_PARAM, domain.ROLE_OFFICER, domain.ROLE_ADMIN))
		{
			own.GET("", r.handler.Customer.GetCustomer)
			own.POST("", r.handler.Customer.UpdateProfile)
			own.GET("/status", r.handler.Customer.GetCustomerStatus)
		}

		customers.GET("", r.handler.Middleware.RequireRole(domain.ROLE_OFFICER, domain.ROLE_ADMIN), r.handler.Customer.ListCustomers)
	}
}

// Файлы висят под тем же /customer/:user_id, поэтому проверка владения и
// подстановка "me" достаются от группы клиента, без отдельного middleware.
func (r *Router) initFileRoutes(api *gin.RouterGroup) {
	files := api.Group("/customer/:user_id/files",
		r.handler.Middleware.RequireAuth(),
		r.handler.Middleware.RequireSelfOrRole(domain.USER_ID_PARAM, domain.ROLE_OFFICER, domain.ROLE_ADMIN))
	{
		files.POST("", r.handler.Filestore.InitUpload)
		files.GET("", r.handler.Filestore.ListFiles)
		files.GET("/required", r.handler.Filestore.HasRequiredFiles)

		file := files.Group("/:file_id")
		{
			file.POST("/confirm", r.handler.Filestore.ConfirmUpload)
			file.GET("/url", r.handler.Filestore.GetDownloadURL)
		}
	}
}
