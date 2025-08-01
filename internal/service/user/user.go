// Package service 提供用户相关的业务逻辑服务
package service

import (
	"errors"

	authModel "github.com/lin-snow/ech0/internal/model/auth"
	commonModel "github.com/lin-snow/ech0/internal/model/common"
	settingModel "github.com/lin-snow/ech0/internal/model/setting"
	model "github.com/lin-snow/ech0/internal/model/user"
	repository "github.com/lin-snow/ech0/internal/repository/user"
	settingService "github.com/lin-snow/ech0/internal/service/setting"
	cryptoUtil "github.com/lin-snow/ech0/internal/util/crypto"
	jwtUtil "github.com/lin-snow/ech0/internal/util/jwt"
)

// UserService 用户服务结构体，提供用户相关的业务逻辑处理
type UserService struct {
	userRepository repository.UserRepositoryInterface     // 用户数据层接口
	settingService settingService.SettingServiceInterface // 系统设置数据层接口
}

// NewUserService 创建并返回新的用户服务实例
//
// 参数:
//   - userRepository: 用户数据层接口实现
//   - settingService: 系统设置数据层接口实现
//
// 返回:
//   - UserServiceInterface: 用户服务接口实现
func NewUserService(userRepository repository.UserRepositoryInterface, settingService settingService.SettingServiceInterface) UserServiceInterface {
	return &UserService{
		userRepository: userRepository,
		settingService: settingService,
	}
}

// Login 用户登录验证
// 验证用户名和密码，成功后生成JWT token
//
// 参数:
//   - loginDto: 登录数据传输对象，包含用户名和密码
//
// 返回:
//   - string: 生成的JWT token
//   - error: 登录过程中的错误信息
func (userService *UserService) Login(loginDto *authModel.LoginDto) (string, error) {
	// 合法性校验
	if loginDto.Username == "" || loginDto.Password == "" {
		return "", errors.New(commonModel.USERNAME_OR_PASSWORD_NOT_BE_EMPTY)
	}

	// 将密码进行 MD5 加密
	loginDto.Password = cryptoUtil.MD5Encrypt(loginDto.Password)

	// 检查用户是否存在
	user, err := userService.userRepository.GetUserByUsername(loginDto.Username)
	if err != nil {
		return "", errors.New(commonModel.USER_NOTFOUND)
	}

	// 进行密码验证,查看外界传入的密码是否与数据库一致
	if user.Password != loginDto.Password {
		return "", errors.New(commonModel.PASSWORD_INCORRECT)
	}

	// 生成 Token
	token, err := jwtUtil.GenerateToken(jwtUtil.CreateClaims(user))
	if err != nil {
		return "", err
	}

	return token, nil
}

// Register 用户注册
// 注册新用户，包括用户数量限制检查、注册权限检查等
// 第一个注册的用户自动设置为系统管理员
//
// 参数:
//   - registerDto: 注册数据传输对象，包含用户名和密码
//
// 返回:
//   - error: 注册过程中的错误信息
func (userService *UserService) Register(registerDto *authModel.RegisterDto) error {
	// 检查用户数量是否超过限制
	users, err := userService.userRepository.GetAllUsers()
	if err != nil {
		return err
	}
	if len(users) > authModel.MAX_USER_COUNT {
		return errors.New(commonModel.USER_COUNT_EXCEED_LIMIT)
	}

	// 将密码进行 MD5 加密
	registerDto.Password = cryptoUtil.MD5Encrypt(registerDto.Password)

	newUser := model.User{
		Username: registerDto.Username,
		Password: registerDto.Password,
		IsAdmin:  false,
	}

	// 检查用户是否已经存在
	user, err := userService.userRepository.GetUserByUsername(newUser.Username)
	if err == nil && user.ID != model.USER_NOT_EXISTS_ID {
		return errors.New(commonModel.USERNAME_HAS_EXISTS)
	}

	// 检查是否该系统第一次注册用户
	if len(users) == 0 {
		// 第一个注册的用户为系统管理员
		newUser.IsAdmin = true
	}

	// 检查是否开放注册
	var setting settingModel.SystemSetting
	if err := userService.settingService.GetSetting(&setting); err != nil {
		return err
	}
	if len(users) != 0 && !setting.AllowRegister {
		return errors.New(commonModel.USER_REGISTER_NOT_ALLOW)
	}

	if err := userService.userRepository.CreateUser(&newUser); err != nil {
		return err
	}

	return nil
}

// UpdateUser 更新用户信息
// 只有管理员可以更新用户信息，支持更新用户名、密码和头像
//
// 参数:
//   - userid: 执行更新操作的用户ID（必须为管理员）
//   - userdto: 用户信息数据传输对象，包含要更新的用户信息
//
// 返回:
//   - error: 更新过程中的错误信息
func (userService *UserService) UpdateUser(userid uint, userdto model.UserInfoDto) error {
	// 检查执行操作的用户是否为管理员
	user, err := userService.userRepository.GetUserByID(int(userid))
	if err != nil {
		return err
	}
	if !user.IsAdmin {
		return errors.New(commonModel.NO_PERMISSION_DENIED)
	}

	// 检查是否需要更新用户名
	if userdto.Username != "" && userdto.Username != user.Username {
		// 检查用户名是否已存在
		existingUser, _ := userService.userRepository.GetUserByUsername(userdto.Username)
		if existingUser.ID != model.USER_NOT_EXISTS_ID {
			return errors.New(commonModel.USERNAME_ALREADY_EXISTS)
		}
		user.Username = userdto.Username
	}

	// 检查是否需要更新密码
	if userdto.Password != "" && cryptoUtil.MD5Encrypt(userdto.Password) != user.Password {
		// 检查密码是否为空
		if userdto.Password == "" {
			return errors.New(commonModel.USERNAME_OR_PASSWORD_NOT_BE_EMPTY)
		}
		// 更新密码
		user.Password = cryptoUtil.MD5Encrypt(userdto.Password)
	}

	// 检查是否需要更新头像
	if userdto.Avatar != "" && userdto.Avatar != user.Avatar {
		// 更新头像
		user.Avatar = userdto.Avatar
	}
	// 更新用户信息
	if err := userService.userRepository.UpdateUser(&user); err != nil {
		return err
	}

	return nil
}

// UpdateUserAdmin 更新用户的管理员权限
// 只有系统管理员、管理员可以修改其他用户的管理员权限，不能修改自己和系统管理员的权限
//
// 参数:
//   - userid: 执行操作的用户ID（必须为管理员）
//   - id: 要修改权限的用户ID
//
// 返回:
//   - error: 更新过程中的错误信息
func (userService *UserService) UpdateUserAdmin(userid uint, id uint) error {
	// 检查执行操作的用户是否为管理员
	user, err := userService.userRepository.GetUserByID(int(userid))
	if err != nil {
		return err
	}
	if !user.IsAdmin {
		return errors.New(commonModel.NO_PERMISSION_DENIED)
	}

	// 检查要修改权限的用户是否存在
	user, err = userService.userRepository.GetUserByID(int(id))
	if err != nil {
		return err
	}

	// 检查系统管理员信息
	sysadmin, err := userService.GetSysAdmin()
	if err != nil {
		return err
	}

	// 检查是否尝试修改自己或系统管理员的权限
	if userid == user.ID || id == sysadmin.ID {
		return errors.New(commonModel.INVALID_PARAMS_BODY)
	}

	user.IsAdmin = !user.IsAdmin

	// 更新用户信息
	if err := userService.userRepository.UpdateUser(&user); err != nil {
		return err
	}

	return nil
}

// GetAllUsers 获取所有用户列表
// 返回除系统管理员外的所有用户，并移除密码信息
//
// 返回:
//   - []model.User: 用户列表（不包含密码信息）
//   - error: 获取过程中的错误信息
func (userService *UserService) GetAllUsers() ([]model.User, error) {
	allures, err := userService.userRepository.GetAllUsers()
	if err != nil {
		return nil, err
	}

	sysadmin, err := userService.GetSysAdmin()
	if err != nil {
		return nil, err
	}

	// 处理用户信息(去掉管理员用户)
	for i := range allures {
		if allures[i].ID == sysadmin.ID {
			allures = append(allures[:i], allures[i+1:]...)
			break
		}
	}

	// 处理用户信息(去掉密码)
	for i := range allures {
		allures[i].Password = ""
	}

	return allures, nil
}

// GetSysAdmin 获取系统管理员信息
//
// 返回:
//   - model.User: 系统管理员用户信息
//   - error: 获取过程中的错误信息
func (userService *UserService) GetSysAdmin() (model.User, error) {
	sysadmin, err := userService.userRepository.GetSysAdmin()
	if err != nil {
		return model.User{}, err
	}

	return sysadmin, nil
}

// DeleteUser 删除用户
// 只有管理员可以删除用户，不能删除自己和系统管理员
//
// 参数:
//   - userid: 执行删除操作的用户ID（必须为管理员）
//   - id: 要删除的用户ID
//
// 返回:
//   - error: 删除过程中的错误信息
func (userService *UserService) DeleteUser(userid, id uint) error {
	// 检查执行操作的用户是否为管理员
	user, err := userService.userRepository.GetUserByID(int(userid))
	if err != nil {
		return err
	}
	if !user.IsAdmin {
		return errors.New(commonModel.NO_PERMISSION_DENIED)
	}

	// 检查要删除的用户是否存在
	user, err = userService.userRepository.GetUserByID(int(id))
	if err != nil {
		return err
	}

	sysadmin, err := userService.GetSysAdmin()
	if err != nil {
		return err
	}

	if userid == user.ID || id == sysadmin.ID {
		return errors.New(commonModel.INVALID_PARAMS_BODY)
	}

	if err := userService.userRepository.DeleteUser(id); err != nil {
		return err
	}

	return nil
}

// GetUserByID 根据用户ID获取用户信息
//
// 参数:
//   - userId: 用户ID
//
// 返回:
//   - model.User: 用户信息
//   - error: 获取过程中的错误信息
func (userService *UserService) GetUserByID(userId int) (model.User, error) {
	return userService.userRepository.GetUserByID(userId)
}
