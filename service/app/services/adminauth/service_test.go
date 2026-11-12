package adminauth

import (
	"context"
	"errors"
	"testing"

	"suki/xqyun/app/models"
	"suki/xqyun/app/services/auth"
)

type fakeAuthenticator struct {
	result   *auth.LoginResult
	err      error
	received auth.Credentials
	calls    int
}

func (f *fakeAuthenticator) Login(_ context.Context, credentials auth.Credentials) (*auth.LoginResult, error) {
	f.calls++
	f.received = credentials
	return f.result, f.err
}

func stringPtr(value string) *string {
	return &value
}

func TestLoginSuccess(t *testing.T) {
	fake := &fakeAuthenticator{
		result: &auth.LoginResult{
			Pair: &auth.TokenPair{ExpiresIn: 900},
			User: &models.User{
				ID:       7,
				Username: "admin",
				Name:     "小阙",
				Email:    stringPtr("admin@example.com"),
				Mobile:   stringPtr("13800000000"),
				Avatar:   stringPtr("data:image/webp;base64,abc"),
			},
		},
	}

	response, err := NewLoginService(fake).Login(context.Background(), LoginInput{
		Username:  " admin ",
		Password:  "123456",
		UserAgent: "test-agent",
		IP:        "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Login() 返回错误：%v", err)
	}
	if fake.calls != 1 {
		t.Fatalf("认证服务调用次数 = %d，期望 1", fake.calls)
	}
	if fake.received.Username != "admin" {
		t.Fatalf("用户名未去除空格：%q", fake.received.Username)
	}
	if fake.received.Password != "123456" ||
		fake.received.UserAgent != "test-agent" ||
		fake.received.IP != "127.0.0.1" {
		t.Fatalf("认证参数未正确传递：%#v", fake.received)
	}

	if response.ExpiresIn != 900 {
		t.Fatalf("ExpiresIn 不正确：%#v", response)
	}
	if response.User.ID != "7" ||
		response.User.Username != "admin" ||
		response.User.Name != "小阙" ||
		response.User.Email != "admin@example.com" ||
		response.User.Mobile != "13800000000" ||
		response.User.Avatar != "data:image/webp;base64,abc" {
		t.Fatalf("用户信息不正确：%#v", response.User)
	}
	if response.User.Roles == nil || len(response.User.Roles) != 0 {
		t.Fatalf("roles 应为空数组：%#v", response.User.Roles)
	}
	if response.User.Permissions == nil || len(response.User.Permissions) != 0 {
		t.Fatalf("permissions 应为空数组：%#v", response.User.Permissions)
	}
}

func TestLoginRejectsEmptyInput(t *testing.T) {
	inputs := []LoginInput{
		{},
		{Username: "admin"},
		{Username: "   ", Password: "123456"},
	}

	for _, input := range inputs {
		if _, err := NewLoginService(&fakeAuthenticator{}).Login(context.Background(), input); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("输入 %#v 应返回 ErrInvalidInput，实际 %v", input, err)
		}
	}
}

func TestLoginMapsAuthErrors(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want *Error
	}{
		{name: "invalid credentials", err: auth.ErrInvalidCredentials, want: ErrInvalidCredentials},
		{name: "account disabled", err: auth.ErrAccountDisabled, want: ErrAccountDisabled},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewLoginService(&fakeAuthenticator{err: testCase.err}).Login(context.Background(), LoginInput{
				Username: "admin",
				Password: "123456",
			})
			if !errors.Is(err, testCase.want) {
				t.Fatalf("错误映射不正确：期望 %v，实际 %v", testCase.want, err)
			}
		})
	}
}

func TestLoginPassesThroughUnknownError(t *testing.T) {
	unknown := errors.New("database unavailable")
	if _, err := NewLoginService(&fakeAuthenticator{err: unknown}).Login(context.Background(), LoginInput{
		Username: "admin",
		Password: "123456",
	}); !errors.Is(err, unknown) {
		t.Fatalf("未知错误应原样返回，实际 %v", err)
	}
}

func TestLoginRejectsNilResult(t *testing.T) {
	if _, err := NewLoginService(&fakeAuthenticator{}).Login(context.Background(), LoginInput{
		Username: "admin",
		Password: "123456",
	}); err == nil {
		t.Fatal("认证服务返回空结果时应报错")
	}
}

func TestLoginRejectsNilAuthenticator(t *testing.T) {
	if _, err := NewLoginService(nil).Login(context.Background(), LoginInput{
		Username: "admin",
		Password: "123456",
	}); err == nil {
		t.Fatal("未注入认证服务时应报错")
	}
}

type fakeLogouter struct {
	err   error
	calls int
}

func (f *fakeLogouter) Logout(_ context.Context) error {
	f.calls++
	return f.err
}

func TestLogoutCallsService(t *testing.T) {
	logouter := &fakeLogouter{}
	if err := NewLogoutService(logouter).Logout(context.Background()); err != nil {
		t.Fatalf("Logout() 返回错误：%v", err)
	}
	if logouter.calls != 1 {
		t.Fatalf("退出登录调用次数 = %d，期望 1", logouter.calls)
	}
}

func TestLogoutPassesThroughError(t *testing.T) {
	unknown := errors.New("redis unavailable")
	if err := NewLogoutService(&fakeLogouter{err: unknown}).Logout(context.Background()); !errors.Is(err, unknown) {
		t.Fatalf("错误应原样返回，实际 %v", err)
	}
}

func TestLogoutRejectsNilService(t *testing.T) {
	if err := NewLogoutService(nil).Logout(context.Background()); err == nil {
		t.Fatal("未注入退出登录服务时应报错")
	}
}
