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
	r.initKycRoutes(api)
	r.initAccountRoutes(api)
	r.initLedgerRoutes(api)

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

// Заявка клиента висит под тем же /customer/:user_id, что и анкета с файлами:
// проверка владения и подстановка "me" достаются от общей группы. Очередь
// офицера - отдельная ветка /kyc, она не про конкретного клиента.
func (r *Router) initKycRoutes(api *gin.RouterGroup) {
	own := api.Group("/customer/:user_id/kyc",
		r.handler.Middleware.RequireAuth(),
		r.handler.Middleware.RequireSelfOrRole(domain.USER_ID_PARAM, domain.ROLE_OFFICER, domain.ROLE_ADMIN))
	{
		own.POST("", r.handler.Kyc.SubmitApplication)
		own.GET("", r.handler.Kyc.GetApplication)
	}

	queue := api.Group("/kyc/applications",
		r.handler.Middleware.RequireAuth(),
		r.handler.Middleware.RequireRole(domain.ROLE_OFFICER, domain.ROLE_ADMIN))
	{
		queue.GET("", r.handler.Kyc.ListApplications)

		application := queue.Group("/:application_id")
		{
			application.GET("", r.handler.Kyc.GetApplicationByID)
			application.POST("/claim", r.handler.Kyc.ClaimApplication)
			application.POST("/approve", r.handler.Kyc.ApproveApplication)
			application.POST("/reject", r.handler.Kyc.RejectApplication)
			application.POST("/documents", r.handler.Kyc.RequestDocuments)
		}
	}

	sanctions := api.Group("/kyc/sanctions",
		r.handler.Middleware.RequireAuth(),
		r.handler.Middleware.RequireRole(domain.ROLE_OFFICER, domain.ROLE_ADMIN))
	{
		sanctions.GET("", r.handler.Kyc.ListSanctions)
		sanctions.POST("", r.handler.Kyc.AddSanctionEntry)
	}
}

// Счета клиента висят под /customer/:user_id ради проверки владения; блокировка - отдельная ветка офицера.
func (r *Router) initAccountRoutes(api *gin.RouterGroup) {
	own := api.Group("/customer/:user_id/accounts",
		r.handler.Middleware.RequireAuth(),
		r.handler.Middleware.RequireSelfOrRole(domain.USER_ID_PARAM, domain.ROLE_OFFICER, domain.ROLE_ADMIN))
	{
		own.POST("", r.handler.Account.OpenAccount)
		own.GET("", r.handler.Account.ListAccounts)

		account := own.Group("/:account_id")
		{
			account.GET("", r.handler.Account.GetAccount)
			account.GET("/balance", r.handler.Account.GetBalance)
			account.GET("/transactions", r.handler.Ledger.ListTransactions)
			account.POST("/close", r.handler.Account.CloseAccount)
		}
	}

	officer := api.Group("/accounts/:account_id",
		r.handler.Middleware.RequireAuth(),
		r.handler.Middleware.RequireRole(domain.ROLE_OFFICER, domain.ROLE_ADMIN))
	{
		officer.POST("/block", r.handler.Account.BlockAccount)
		officer.POST("/unblock", r.handler.Account.UnblockAccount)
		officer.POST("/deposit", r.handler.Ledger.Deposit)
		officer.GET("/history", r.handler.Account.ListStatusHistory)
	}
}

// Перевод идёт от имени владельца счёта-источника, id инициатора берётся из пути после проверки владения.
func (r *Router) initLedgerRoutes(api *gin.RouterGroup) {
	api.POST("/customer/:user_id/transfers",
		r.handler.Middleware.RequireAuth(),
		r.handler.Middleware.RequireSelfOrRole(domain.USER_ID_PARAM, domain.ROLE_OFFICER, domain.ROLE_ADMIN),
		r.handler.Ledger.Transfer)

	api.GET("/transactions/:transaction_id",
		r.handler.Middleware.RequireAuth(),
		r.handler.Middleware.RequireRole(domain.ROLE_OFFICER, domain.ROLE_ADMIN),
		r.handler.Ledger.GetTransaction)
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
