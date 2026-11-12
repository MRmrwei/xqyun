/**
 * 当前管理员资料与密码接口封装。
 *
 * 路径相对于 API_BASE_URL（默认 /api/v1），请求层自动携带 HttpOnly 会话 Cookie。
 * 用户 ID 由后端从登录会话解析，前端不传。
 */

import { api } from '@/services/api'

/** PUT /api/v1/admin/user/profile 的响应 data。 */
export interface ApiAdminProfile {
  id: string
  username: string
  name: string
  email: string
  mobile: string
  avatar: string
}

export interface ApiUpdateProfileRequest {
  name: string
  email: string
  mobile: string
  avatar: string
}

/** GET /api/v1/admin/me 的响应 data；用于校验会话并同步最新资料。 */
export function getCurrentAdmin(): Promise<ApiAdminProfile> {
  return api.get<ApiAdminProfile>('/admin/me')
}

export function updateProfile(body: ApiUpdateProfileRequest): Promise<ApiAdminProfile> {
  return api.put<ApiAdminProfile>('/admin/user/profile', body)
}

/** 修改当前管理员密码；成功后下次登录使用新密码。 */
export function changeAdminPassword(
  currentPassword: string,
  newPassword: string
): Promise<void> {
  return api.put<void>('/admin/user/password', {
    current_password: currentPassword,
    new_password: newPassword
  })
}
