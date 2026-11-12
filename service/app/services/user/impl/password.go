// Package impl 提供 user 包业务接口的具体实现。
package impl

import (
	"context"
	"errors"
	"time"
	"unicode/utf8"

	userrepository "suki/xqyun/app/repository/user"
	"suki/xqyun/app/services/auth"
	"suki/xqyun/app/services/user"
)

// minPasswordRunes 是新密码的最小字符数（中文按字符计算）。
const minPasswordRunes = 8

// PasswordChanger 是 user.PasswordChanger 的实现。
type PasswordChanger struct {
	users userrepository.UserRepository
	now   func() time.Time
}

var _ user.PasswordChanger = (*PasswordChanger)(nil)

// NewPasswordChanger 创建修改密码用例。
func NewPasswordChanger(users userrepository.UserRepository) *PasswordChanger {
	return &PasswordChanger{
		users: users,
		now:   time.Now,
	}
}

// ChangePassword 校验当前密码后重新哈希并保存新密码。
func (s *PasswordChanger) ChangePassword(
	ctx context.Context,
	input user.ChangePasswordInput,
) error {
	if input.UserID == 0 {
		return user.ErrUserNotFound
	}

	account, err := s.users.FindByID(ctx, input.UserID)
	if errors.Is(err, userrepository.ErrNotFound) {
		return user.ErrUserNotFound
	}
	if err != nil {
		return err
	}

	if !auth.VerifyPassword(input.CurrentPassword, account.PasswordHash) {
		return user.ErrCurrentPasswordIncorrect
	}
	if utf8.RuneCountInString(input.NewPassword) < minPasswordRunes {
		return user.ErrPasswordTooShort
	}

	hashed, err := auth.HashPassword(input.NewPassword)
	if err != nil {
		if errors.Is(err, auth.ErrPasswordTooLong) {
			return user.ErrPasswordTooLong
		}
		return err
	}

	if err := s.users.UpdatePassword(ctx, input.UserID, hashed, s.now().Unix()); err != nil {
		if errors.Is(err, userrepository.ErrNotFound) {
			return user.ErrUserNotFound
		}
		return err
	}

	return nil
}
