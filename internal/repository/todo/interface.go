package repository

import model "github.com/lin-snow/ech0/internal/model/todo"

type TodoRepositoryInterface interface {
	// GetTodosByUserID 根据用户ID获取待办事项
	GetTodosByUserID(userid uint) ([]model.Todo, error)

	// CreateTodo 创建一个新的待办事项
	CreateTodo(todo *model.Todo) error

	// GetTodoByID 根据ID获取待办事项
	GetTodoByID(todoID int64) (*model.Todo, error)

	// UpdateTodo 更新待办事项
	UpdateTodo(todo *model.Todo) error

	// DeleteTodo 删除待办事项
	DeleteTodo(id int64) error
}
