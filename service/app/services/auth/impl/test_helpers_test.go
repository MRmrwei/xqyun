package impl

import (
	"context"

	"suki/xqyun/app/models"
	userrepository "suki/xqyun/app/repository/user"
)

type fakeUsers struct {
	users map[uint64]*models.User
}

var _ userrepository.UserRepository = (*fakeUsers)(nil)

func newFakeUsers() *fakeUsers {
	return &fakeUsers{users: map[uint64]*models.User{}}
}

func (f *fakeUsers) FindByUsername(_ context.Context, username string) (*models.User, error) {
	for _, user := range f.users {
		if user.Username == username {
			copy := *user
			return &copy, nil
		}
	}
	return nil, userrepository.ErrNotFound
}

func (f *fakeUsers) FindByID(_ context.Context, id uint64) (*models.User, error) {
	user, ok := f.users[id]
	if !ok {
		return nil, userrepository.ErrNotFound
	}
	copy := *user
	return &copy, nil
}

func (f *fakeUsers) UpdateLoginState(context.Context, uint64, int64, *string) error {
	return nil
}

func (f *fakeUsers) UpdateProfile(
	context.Context,
	uint64,
	string,
	*string,
	*string,
	*string,
	int64,
) error {
	return nil
}

func (f *fakeUsers) UpdatePassword(context.Context, uint64, string, int64) error {
	return nil
}
