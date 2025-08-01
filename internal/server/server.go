package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lin-snow/ech0/internal/cache"
	"github.com/lin-snow/ech0/internal/config"
	"github.com/lin-snow/ech0/internal/database"
	"github.com/lin-snow/ech0/internal/di"
	commonModel "github.com/lin-snow/ech0/internal/model/common"
	"github.com/lin-snow/ech0/internal/router"
	errUtil "github.com/lin-snow/ech0/internal/util/err"
)

// Server 服务器结构体，包含Gin引擎
type Server struct {
	GinEngine  *gin.Engine
	httpServer *http.Server // 用于优雅停止服务器
}

// New 创建一个新的服务器实例
func New() *Server {
	return &Server{}
}

// Init 初始化服务器
func (s *Server) Init() {
	// Mode
	if config.Config.Server.Mode == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Gin Engine
	s.GinEngine = gin.New()

	// Database
	database.InitDatabase()

	// Cache
	cacheFactory := cache.NewCacheFactory()

	// Handlers
	handlers, err := di.BuildHandlers(database.DB, cacheFactory)
	if err != nil {
		errUtil.HandlePanicError(&commonModel.ServerError{
			Msg: commonModel.INIT_HANDLERS_PANIC,
			Err: err,
		})
	}

	// Router
	router.SetupRouter(s.GinEngine, handlers)
}

// Start 异步启动服务器
func (s *Server) Start() {
	port := config.Config.Server.Port
	PrintGreetings(port)

	s.httpServer = &http.Server{
		Addr:    ":" + port,
		Handler: s.GinEngine,
	}

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errUtil.HandlePanicError(&commonModel.ServerError{
				Msg: commonModel.GIN_RUN_FAILED,
				Err: err,
			})
		}
	}()
	fmt.Println("🚀 Ech0 Server已启动，监听端口", port)
}

// Stop 优雅停止服务器
func (s *Server) Stop(ctx context.Context) error {
	// 使用传入的 context，如果没有则创建默认的 5 秒超时
	shutdownCtx := ctx
	var cancel context.CancelFunc

	if ctx == nil {
		shutdownCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}

	if s.httpServer == nil {
		fmt.Println("⚠️ HTTP 服务器未启动，无需关闭")
		return nil
	}

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return err
	}

	return nil
}
