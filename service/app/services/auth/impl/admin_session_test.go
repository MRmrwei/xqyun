package impl

import (
	"context"
	"errors"
	"testing"
	"time"

	"suki/xqyun/app/models"
	userrepository "suki/xqyun/app/repository/user"
	"suki/xqyun/app/services/auth"
)

const testSessionTTL = 8 * time.Hour

// fakeSessionStore 模拟 hertz-contrib/sessions 的会话读写。
type fakeSessionStore struct {
	values  map[string]any
	saved   int
	cleared bool
	saveErr error
}

var _ auth.SessionStore = (*fakeSessionStore)(nil)

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{values: map[string]any{}}
}

func (f *fakeSessionStore) Set(key string, value any) {
	f.values[key] = value
}

func (f *fakeSessionStore) Get(key string) (any, bool) {
	value, ok := f.values[key]
	return value, ok
}

func (f *fakeSessionStore) Clear() {
	f.values = map[string]any{}
	f.cleared = true
}

func (f *fakeSessionStore) Save() error {
	f.saved++
	return f.saveErr
}

func newTestAdminSession(
	users userrepository.UserRepository,
	store auth.SessionStore,
	now time.Time,
) *AdminSession {
	session := NewAdminSession(users, store, AdminSessionConfig{TTL: testSessionTTL})
	session.now = func() time.Time { return now }
	return session
}

func newTestUser(t *testing.T) *models.User {
	t.Helper()

	hashed, err := auth.HashPassword("123456")
	if err != nil {
		t.Fatalf("生成测试密码失败：%v", err)
	}
	return &models.User{
		ID:           7,
		Username:     "admin",
		Name:         "小阙",
		PasswordHash: hashed,
		Status:       1,
	}
}

func TestSessionLoginStoresIdentity(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	users := newFakeUsers()
	users.users[7] = newTestUser(t)
	store := newFakeSessionStore()
	session := newTestAdminSession(users, store, now)

	result, err := session.Login(context.Background(), auth.Credentials{
		Username:  "admin",
		Password:  "123456",
		UserAgent: "test-agent",
		IP:        "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Login() 返回错误：%v", err)
	}
	if result == nil || result.Pair == nil || result.User == nil {
		t.Fatal("Login() 结果不完整")
	}
	if result.Pair.AccessToken != "" || result.Pair.RefreshToken != "" {
		t.Fatal("Session 模式不应返回令牌")
	}
	if result.Pair.ExpiresIn != int(testSessionTTL/time.Second) {
		t.Fatalf("ExpiresIn = %d，期望 %d", result.Pair.ExpiresIn, int(testSessionTTL/time.Second))
	}
	if result.User.ID != 7 {
		t.Fatalf("用户 ID = %d，期望 7", result.User.ID)
	}

	userID, ok := store.values[auth.SessionKeyAdminUserID]
	if !ok || userID != uint64(7) {
		t.Fatalf("会话未写入用户 ID：%#v", store.values)
	}
	if username, ok := store.values[auth.SessionKeyAdminUsername]; !ok || username != "admin" {
		t.Fatalf("会话未写入用户名：%#v", store.values)
	}
	if store.saved == 0 {
		t.Fatal("登录后应保存会话")
	}
}

func TestSessionLogoutClearsStore(t *testing.T) {
	store := newFakeSessionStore()
	store.Set(auth.SessionKeyAdminUserID, uint64(7))
	service := NewAdminSessionLogout(store)

	if err := service.Logout(context.Background()); err != nil {
		t.Fatalf("Logout() 返回错误：%v", err)
	}
	if !store.cleared {
		t.Fatal("Logout() 应清空会话")
	}
	if len(store.values) != 0 {
		t.Fatalf("会话仍有残留值：%#v", store.values)
	}
	if store.saved == 0 {
		t.Fatal("清空后应保存，确保删除 Cookie 与服务端记录")
	}

	// 重复退出保持幂等。
	if err := service.Logout(context.Background()); err != nil {
		t.Fatalf("重复 Logout() 应成功，实际 %v", err)
	}
}

func TestSessionLoginRejectsBadCredentials(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)

	cases := []struct {
		name     string
		password string
		status   int8
		want     error
	}{
		{name: "wrong password", password: "wrong", status: 1, want: auth.ErrInvalidCredentials},
		{name: "disabled user", password: "123456", status: 0, want: auth.ErrAccountDisabled},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			users := newFakeUsers()
			user := newTestUser(t)
			user.Status = testCase.status
			users.users[7] = user
			session := newTestAdminSession(users, newFakeSessionStore(), now)

			if _, err := session.Login(context.Background(), auth.Credentials{
				Username: "admin",
				Password: testCase.password,
			}); !errors.Is(err, testCase.want) {
				t.Fatalf("应返回 %v，实际 %v", testCase.want, err)
			}
		})
	}
}

