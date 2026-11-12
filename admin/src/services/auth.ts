/**
 * 管理后台登录与当前管理员资料。
 *
 * 已接 Go API：
 * - POST /api/v1/admin/auth/login
 * - POST /api/v1/admin/auth/logout
 * - GET  /api/v1/admin/me
 * - PUT  /api/v1/admin/user/profile
 * - PUT  /api/v1/admin/user/password
 *
 * 登录会话由后端写入 HttpOnly Cookie，前端不保存任何令牌；
 * localStorage 只缓存用户展示信息，用于刷新页面时恢复界面。
 */
import { ref } from 'vue'
import type { AdminLoginResponse, AdminProfile, AdminUser } from '@/types/admin'
import { ApiError, api, configureUnauthorizedHandler } from '@/services/api'
import {
  changeAdminPassword,
  getCurrentAdmin,
  updateProfile,
  type ApiAdminProfile
} from '@/services/userApi'

export interface AuthSession {
  username: string
  name: string
  loggedInAt: string
  user: AdminUser
}

export interface ChangePasswordResult {
  ok: boolean
  message: string
}

/** 账号页「恢复默认资料」使用的占位值；登录账号仍以 Go 后端为准。 */
export const demoCredentials = {
  username: 'admin',
  name: '小阙'
} as const

export const AUTH_STORAGE_KEY = 'xiaoque-admin-auth-v1'

const MIN_PASSWORD_CHARS = 8
const MAX_PASSWORD_BYTES = 72
const UNAUTHORIZED_CODES = new Set([401, 40100, 40101])

const DEFAULT_PROFILE: AdminProfile = {
  username: demoCredentials.username,
  name: demoCredentials.name,
  email: '',
  mobile: '',
  avatar: ''
}

let memorySession: AuthSession | null = null
let memoryProfile: AdminProfile | null = null
/** 当前页面生命周期内是否已经向后端确认过会话仍然有效。 */
let sessionVerified = false

function readStorage<T>(key: string): T | null {
  try {
    const raw = localStorage.getItem(key)
    if (!raw) return null
    return JSON.parse(raw) as T
  } catch {
    return null
  }
}

function writeStorage(key: string, value: unknown): boolean {
  try {
    localStorage.setItem(key, JSON.stringify(value))
    return true
  } catch {
    return false
  }
}

function removeStorage(key: string): void {
  try {
    localStorage.removeItem(key)
  } catch {
    // ignore unavailable storage
  }
}

function readSession(): AuthSession | null {
  if (memorySession) return memorySession

  const parsed = readStorage<Partial<AuthSession>>(AUTH_STORAGE_KEY)
  if (!parsed || !parsed.user || typeof parsed.user.username !== 'string') {
    return null
  }

  const user = parsed.user
  const session: AuthSession = {
    username:
      typeof parsed.username === 'string' && parsed.username
        ? parsed.username
        : user.username,
    name: typeof parsed.name === 'string' && parsed.name ? parsed.name : user.name,
    loggedInAt:
      typeof parsed.loggedInAt === 'string' && parsed.loggedInAt
        ? parsed.loggedInAt
        : new Date().toISOString(),
    user
  }
  memorySession = session
  return session
}

/** 管理员资料以登录会话中的用户信息为准，不再单独存 localStorage。 */
function readProfile(): AdminProfile {
  if (memoryProfile) return memoryProfile

  const user = readSession()?.user
  const profile: AdminProfile = user
    ? {
        username: user.username,
        name: user.name || user.username,
        email: user.email,
        mobile: user.mobile,
        avatar: user.avatar
      }
    : { ...DEFAULT_PROFILE }

  memoryProfile = profile
  return profile
}

/** 当前登录会话；侧边栏等组件用它实时显示管理员名称。 */
export const sessionRef = ref<AuthSession | null>(readSession())

/** 当前管理员资料；侧边栏头像与账号页共用。 */
export const profileRef = ref<AdminProfile>(readProfile())

// 401 时清除本地会话并跳转登录页。
configureUnauthorizedHandler(() => {
  clearSession()
  if (typeof window === 'undefined') return
  const current = window.location.hash.replace(/^#/, '') || '/pages'
  if (!current.startsWith('/login')) {
    window.location.hash = `#/login?redirect=${encodeURIComponent(current)}`
  }
})

function passwordByteLength(value: string): number {
  return new TextEncoder().encode(value).length
}

/** 校验新密码长度；返回中文错误提示，通过时返回 null。 */
export function validateNewPassword(value: string): string | null {
  if (!value) return '请输入新密码'
  if (Array.from(value).length < MIN_PASSWORD_CHARS) {
    return `新密码至少 ${MIN_PASSWORD_CHARS} 个字符`
  }
  if (passwordByteLength(value) > MAX_PASSWORD_BYTES) {
    return `新密码不能超过 ${MAX_PASSWORD_BYTES} 字节（中文约 24 个字符）`
  }
  return null
}

/** 登录：调用 Go API，会话写入 HttpOnly Cookie，本地只保存用户信息。 */
export async function login(username: string, password: string): Promise<AuthSession> {
  const account = username.trim()
  if (!account || !password) {
    throw new Error('请输入用户名和密码')
  }

  const data = await api.post<AdminLoginResponse>('/admin/auth/login', {
    username: account,
    password
  })
  if (!data?.user) {
    throw new Error('登录响应格式不正确')
  }

  const user = normalizeUser(data.user, account)
  const session: AuthSession = {
    username: user.username,
    name: user.name,
    loggedInAt: new Date().toISOString(),
    user
  }

  memorySession = session
  sessionRef.value = session
  sessionVerified = true
  writeStorage(AUTH_STORAGE_KEY, session)
  seedProfileFromUser(user)
  return session
}

/** 统一补齐登录 / 当前用户响应里的用户字段，避免后端缺字段导致界面空白。 */
function normalizeUser(raw: AdminUser, fallbackUsername: string): AdminUser {
  return {
    ...raw,
    id: String(raw.id ?? ''),
    username: raw.username || fallbackUsername,
    name: raw.name || raw.username || fallbackUsername,
    email: raw.email || '',
    mobile: raw.mobile || '',
    avatar: raw.avatar || '',
    roles: Array.isArray(raw.roles) ? raw.roles : [],
    permissions: Array.isArray(raw.permissions) ? raw.permissions : []
  }
}

function seedProfileFromUser(user: AdminUser): void {
  const profile: AdminProfile = {
    username: user.username,
    name: user.name,
    email: user.email,
    mobile: user.mobile,
    avatar: user.avatar
  }
  memoryProfile = profile
  profileRef.value = { ...profile }
}

export function getSession(): AuthSession | null {
  return readSession()
}

/**
 * 向后端确认当前 Cookie 会话仍然有效，并同步最新管理员资料。
 * 同一页面生命周期内只校验一次；网络故障时保留本地会话，交给后续请求判断。
 */
export async function validateSession(): Promise<boolean> {
  if (!sessionRef.value) return false
  if (sessionVerified) return true

  try {
    const profile = await getCurrentAdmin()
    applyProfile(profile)
    sessionVerified = true
    return true
  } catch (error) {
    if (error instanceof ApiError && UNAUTHORIZED_CODES.has(error.code)) {
      clearSession()
      return false
    }
    return true
  }
}

/** 用后端资料刷新本地会话中的用户信息。 */
function applyProfile(saved: ApiAdminProfile): void {
  const profile: AdminProfile = {
    username: saved.username,
    name: saved.name,
    email: saved.email,
    mobile: saved.mobile,
    avatar: saved.avatar
  }
  memoryProfile = profile
  profileRef.value = { ...profile }

  const session = sessionRef.value
  if (!session) return

  const updated: AuthSession = {
    ...session,
    username: profile.username,
    name: profile.name,
    user: {
      ...session.user,
      id: saved.id,
      username: profile.username,
      name: profile.name,
      email: profile.email,
      mobile: profile.mobile,
      avatar: profile.avatar
    }
  }
  memorySession = updated
  sessionRef.value = updated
  writeStorage(AUTH_STORAGE_KEY, updated)
}

/** 清除本地登录会话，不发起网络请求。 */
function clearSession(): void {
  memorySession = null
  sessionRef.value = null
  sessionVerified = false
  removeStorage(AUTH_STORAGE_KEY)
}

/**
 * 退出登录：先让服务端删除会话，再清除本地状态。
 * 网络或服务端异常时仍然清除本地状态，避免用户卡在已登录界面。
 */
export async function logout(): Promise<void> {
  try {
    await api.post('/admin/auth/logout', undefined, { timeoutMs: 5_000 })
  } catch {
    // 服务端不可用不影响本地退出；服务端会话会自然过期。
  } finally {
    clearSession()
  }
}

export function getProfile(): AdminProfile {
  return { ...readProfile() }
}

/**
 * 保存显示名称、邮箱、手机号和头像到 Go 后端，并同步右上角用户区。
 * 登录账号固定不可修改。
 */
export async function saveProfile(
  input: Pick<AdminProfile, 'name' | 'email' | 'mobile'> & Partial<Pick<AdminProfile, 'avatar'>>
): Promise<AdminProfile> {
  const name = input.name.trim()
  if (!name) throw new Error('显示名称不能为空')

  const saved = await updateProfile({
    name,
    email: input.email.trim(),
    mobile: input.mobile.trim(),
    avatar: (input.avatar ?? '').trim()
  })

  applyProfile(saved)
  return { ...profileRef.value }
}

/**
 * 修改密码：调用 Go 后端，成功后当前会话被删除，需要重新登录。
 * 其他设备上的会话会在 TTL 到期后失效。
 */
export async function changePassword(
  currentPassword: string,
  newPassword: string
): Promise<ChangePasswordResult> {
  try {
    await changeAdminPassword(currentPassword, newPassword)
    clearSession()
    return { ok: true, message: '密码已修改，请使用新密码重新登录' }
  } catch (error) {
    return {
      ok: false,
      message: error instanceof Error ? error.message : '密码修改失败，请稍后重试'
    }
  }
}
