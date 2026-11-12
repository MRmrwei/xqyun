/**
 * 站点编排（页面内容 / 顶部导航 / 底部导航 / 站点配置）Go API 封装。
 *
 * 路径相对于 API_BASE_URL（默认 /api/v1），请求层自动携带 HttpOnly 登录会话。
 */

import { api, type QueryParams } from '@/services/api'

export type ApiLocale = 'zh-CN' | 'en'
export type ApiPageStatus = 'draft' | 'published'
export type ApiLinkType = 'page' | 'content' | 'external' | 'none'
export type ApiLinkTarget = '_self' | '_blank'
export type ApiMenuCode = 'header' | 'footer' | 'footer_bottom'

export interface ApiSeo {
  title: string
  description: string
  keywords: string[]
}

export interface ApiPageBlock {
  id: string
  type: string
  props: Record<string, unknown>
}

export interface ApiPageTranslation {
  page_id: string
  locale: ApiLocale
  path: string
  name: string
  status: ApiPageStatus
}

export interface ApiPageSummary {
  id: string
  group_id: string
  locale: ApiLocale
  path: string
  name: string
  status: ApiPageStatus
  version: number
  updated_at: number
  updated_by: string
  translations: ApiPageTranslation[]
}

export interface ApiPageDraft {
  id: string
  group_id: string
  locale: ApiLocale
  path: string
  name: string
  status: ApiPageStatus
  version: number
  seo: ApiSeo
  blocks: ApiPageBlock[]
  updated_at: number
}

export interface ApiPageListResponse {
  items: ApiPageSummary[]
  total: number
  page: number
  page_size: number
}

export interface ApiPageListQuery {
  locale?: ApiLocale
  status?: ApiPageStatus
  keyword?: string
  page?: number
  page_size?: number
}

export interface ApiPageCreateRequest {
  locale?: ApiLocale
  path: string
  name: string
  seo?: ApiSeo
  blocks?: ApiPageBlock[]
  copy_from_id?: string
}

export interface ApiPageSaveDraftRequest {
  path?: string
  name?: string
  seo?: ApiSeo
  blocks?: ApiPageBlock[]
}

export interface ApiPageTranslateRequest {
  from_locale?: ApiLocale
  to_locale?: ApiLocale
  path: string
  name?: string
  seo?: ApiSeo
  blocks?: ApiPageBlock[]
}

export interface ApiPagePublishRequest {
  note?: string
}

export interface ApiPagePublishResult {
  version: number
  published_at: number
}

export interface ApiMenuItem {
  id?: string
  label: Record<string, string>
  path?: string
  link_type?: ApiLinkType
  target?: ApiLinkTarget
  visible?: boolean
  sort?: number
  children?: ApiMenuItem[]
}

export interface ApiMenuSummary {
  id: string
  code: ApiMenuCode
  name: string
  description: string
  updated_at: number
}

export interface ApiMenuDetail extends ApiMenuSummary {
  items: ApiMenuItem[]
}

export interface ApiMenuListResponse {
  items: ApiMenuSummary[]
}

export interface ApiMenuSaveRequest {
  name?: string
  items: ApiMenuItem[]
}

export interface ApiSiteSettings {
  locale: ApiLocale
  [key: string]: unknown
}

const ADMIN_PREFIX = '/admin'

export function listPages(query: ApiPageListQuery = {}): Promise<ApiPageListResponse> {
  return api.get<ApiPageListResponse>(`${ADMIN_PREFIX}/pages`, {
    query: query as QueryParams
  })
}

export function getPage(id: string): Promise<ApiPageDraft> {
  return api.get<ApiPageDraft>(`${ADMIN_PREFIX}/pages/${encodeURIComponent(id)}`)
}

export function createPage(body: ApiPageCreateRequest): Promise<ApiPageDraft> {
  return api.post<ApiPageDraft>(`${ADMIN_PREFIX}/pages`, body)
}

export function savePageDraft(
  id: string,
  body: ApiPageSaveDraftRequest
): Promise<ApiPageDraft> {
  return api.put<ApiPageDraft>(`${ADMIN_PREFIX}/pages/${encodeURIComponent(id)}`, body)
}

export function deletePage(id: string): Promise<void> {
  return api.delete<void>(`${ADMIN_PREFIX}/pages/${encodeURIComponent(id)}`)
}

export function translatePage(
  id: string,
  body: ApiPageTranslateRequest
): Promise<ApiPageDraft> {
  return api.post<ApiPageDraft>(
    `${ADMIN_PREFIX}/pages/${encodeURIComponent(id)}/translate`,
    body
  )
}

export function publishPage(
  id: string,
  body: ApiPagePublishRequest = {}
): Promise<ApiPagePublishResult> {
  return api.post<ApiPagePublishResult>(
    `${ADMIN_PREFIX}/pages/${encodeURIComponent(id)}/publish`,
    body
  )
}

export function unpublishPage(
  id: string,
  body: ApiPagePublishRequest = {}
): Promise<void> {
  return api.post<void>(
    `${ADMIN_PREFIX}/pages/${encodeURIComponent(id)}/unpublish`,
    body
  )
}

export function listMenus(): Promise<ApiMenuListResponse> {
  return api.get<ApiMenuListResponse>(`${ADMIN_PREFIX}/menus`)
}

export function getMenu(code: ApiMenuCode): Promise<ApiMenuDetail> {
  return api.get<ApiMenuDetail>(`${ADMIN_PREFIX}/menus/${code}`)
}

export function saveMenu(
  code: ApiMenuCode,
  body: ApiMenuSaveRequest
): Promise<ApiMenuDetail> {
  return api.put<ApiMenuDetail>(`${ADMIN_PREFIX}/menus/${code}`, body)
}

export function getSettings(locale: ApiLocale = 'zh-CN'): Promise<ApiSiteSettings> {
  return api.get<ApiSiteSettings>(`${ADMIN_PREFIX}/settings`, { query: { locale } })
}

export function saveSettings(
  locale: ApiLocale,
  values: Record<string, unknown>
): Promise<ApiSiteSettings> {
  return api.put<ApiSiteSettings>(`${ADMIN_PREFIX}/settings`, { locale, ...values })
}
