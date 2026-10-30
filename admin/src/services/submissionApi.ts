/**
 * 表单留言 Go API 封装。
 */

import { api, type QueryParams } from '@/services/api'
import type { ContactSubmissionRecord, SubmissionStatus } from '@/types/admin'
import { formatApiTime } from '@/services/siteMapping'

export type ApiSubmissionLocale = 'zh-CN' | 'en'

export interface ApiSubmission {
  id: string
  form_key: string
  locale: ApiSubmissionLocale
  name: string
  email: string
  phone: string
  company: string
  message: string
  data: Record<string, unknown>
  status: SubmissionStatus
  ip: string
  user_agent: string
  created_at: number
  updated_at: number
}

export interface ApiSubmissionStats {
  total: number
  new: number
  processing: number
  done: number
  spam: number
}

export interface ApiSubmissionListResponse {
  items: ApiSubmission[]
  total: number
  page: number
  page_size: number
  stats: ApiSubmissionStats
}

export interface ApiSubmissionQuery {
  status?: SubmissionStatus
  form_key?: string
  locale?: ApiSubmissionLocale
  keyword?: string
  created_from?: number
  created_to?: number
  page?: number
  page_size?: number
}

export function listSubmissions(
  query: ApiSubmissionQuery = {}
): Promise<ApiSubmissionListResponse> {
  return api.get<ApiSubmissionListResponse>('/admin/submissions', {
    query: query as QueryParams
  })
}

/** 待处理留言数量，供侧边栏红点轮询：只返回一个数字，不带列表。 */
export async function getSubmissionUnreadCount(): Promise<number> {
  const result = await api.get<{ count: number }>('/admin/submissions/unread-count')
  return result.count
}

export function updateSubmissionStatus(
  id: string,
  status: SubmissionStatus
): Promise<ApiSubmission> {
  return api.put<ApiSubmission>(
    `/admin/submissions/${encodeURIComponent(id)}/status`,
    { status }
  )
}

/** 后端留言转后台列表 / 详情模型。 */
export function submissionFromApi(item: ApiSubmission): ContactSubmissionRecord {
  const data: Record<string, string> = {}
  Object.entries(item.data ?? {}).forEach(([key, value]) => {
    if (typeof value === 'string') {
      data[key] = value
      return
    }
    if (value !== null && value !== undefined) {
      data[key] = JSON.stringify(value)
    }
  })

  return {
    id: item.id,
    formKey: item.form_key,
    locale: item.locale === 'en' ? 'en' : 'zh',
    name: item.name,
    email: item.email,
    phone: item.phone,
    company: item.company,
    message: item.message,
    data,
    status: item.status,
    ip: item.ip,
    userAgent: item.user_agent,
    createdAt: formatApiTime(item.created_at),
    updatedAt: formatApiTime(item.updated_at)
  }
}
