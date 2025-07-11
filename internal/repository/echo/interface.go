package repository

import model "github.com/lin-snow/ech0/internal/model/echo"

type EchoRepositoryInterface interface {
	// CreateEcho 创建一个新的 Echo
	CreateEcho(echo *model.Echo) error

	// GetEchosByPage 获取分页的 Echo 列表
	GetEchosByPage(page, pageSize int, search string, showPrivate bool) ([]model.Echo, int64)

	// GetEchosById 根据 ID 获取 Echo
	GetEchosById(id uint) (*model.Echo, error)

	// DeleteEchoById 删除 Echo
	DeleteEchoById(id uint) error

	// GetTodayEchos 获取今天的 Echo 列表
	GetTodayEchos(showPrivate bool) []model.Echo

	// UpdateEcho 更新 Echo
	UpdateEcho(echo *model.Echo) error

	// LikeEcho 点赞 Echo
	LikeEcho(id uint) error
}
