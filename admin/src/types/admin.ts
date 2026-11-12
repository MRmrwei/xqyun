import type { PageBlock, PageRecord, SeoMeta } from '~/domain/types'

export type { Locale, PageBlock, PageRecord, SeoMeta } from '~/domain/types'
export type { SiteCollections, SiteSettings, NewsRecord, ProductRecord, CaseRecord } from '~/domain/types'

export type PageStatus = 'draft' | 'published'
export type EditorSaveState = 'saved' | 'dirty' | 'saving'
export type DeviceMode = 'desktop' | 'tablet' | 'mobile'

export type SubmissionStatus = 'new' | 'processing' | 'done' | 'spam'

/** 后台管理员资料（本地演示，字段与 xqyun_users 对齐）。 */
export interface AdminProfile {
  username: string
  name: string
  email: string
  mobile: string
  /** 自定义头像 data URL；为空时使用名称首字。 */
  avatar: string
}

/** 管理后台登录接口返回的用户信息。 */
export interface AdminUser {
  id: string
  username: string
  name: string
  email: string
  mobile: string
  avatar: string
  roles: string[]
  permissions: string[]
}

/** POST /api/v1/admin/auth/login 的响应 data；登录会话通过 HttpOnly Cookie 下发。 */
export interface AdminLoginResponse {
  expires_in: number
  user: AdminUser
}

/** 后台登录页与登录策略的本地设置。 */
export interface LoginSettings {
  captchaEnabled: boolean
  /** 登录页品牌区底图，data URL 或远程地址；为空时使用黑色底。 */
  backgroundImage: string
  brandName: string
  brandSubName: string
  brandKicker: string
  brandTitle: string
  brandSubtitle: string
}

/** 联系表单提交（本地演示数据，字段与 form_submissions 表对齐）。 */
export interface ContactSubmissionRecord {
  id: string
  formKey: string
  locale: import('~/domain/types').Locale
  name: string
  email: string
  phone: string
  company: string
  message: string
  data: Record<string, string>
  status: SubmissionStatus
  ip: string
  userAgent: string
  createdAt: string
  updatedAt: string
}

/** 本地只持久化仍在演示阶段的登录页设置。 */
export interface PersistedDemo {
  schemaVersion: 1
  savedAt: string
  pages?: PageRecord[]
  loginSettings?: LoginSettings
  settings?: import('~/domain/types').SiteSettings
}

export interface PreviewUpdateMessage {
  type: 'xq-preview:update'
  revision: number
  page: PageRecord
  locale: 'zh' | 'en'
  collections: import('~/domain/types').SiteCollections
  settings: import('~/domain/types').SiteSettings
  /** 点击左侧区块时定位到的区块 ID；token 用于区分同一区块的重复点击。 */
  focusBlockId?: string
  focusToken?: number
}
