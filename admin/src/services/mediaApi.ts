/**
 * 媒体素材接口封装（视频 / 图片）。
 *
 * 上传走 multipart/form-data，请求层自动携带 HttpOnly 登录会话。
 * kind 作为分类字段保留，后续可扩展其他素材类型。
 */

import { api } from '@/services/api'

export type MediaKind = 'video' | 'image'

export interface ApiMediaItem {
  id: string
  kind: string
  url: string
  filename: string
  original_name: string
  mime: string
  size: number
  created_at: number
}

export interface ApiMediaListResponse {
  items: ApiMediaItem[]
  total: number
  page: number
  page_size: number
}

/** 视频上传可能耗时较长，单独放宽超时。 */
const UPLOAD_TIMEOUT_MS = 10 * 60 * 1000

export function listMedia(
  kind: MediaKind | 'all' = 'video',
  page = 1,
  pageSize = 60,
  keyword = ''
): Promise<ApiMediaListResponse> {
  return api.get<ApiMediaListResponse>('/admin/media', {
    query: {
      kind: kind === 'all' ? undefined : kind,
      keyword: keyword.trim() || undefined,
      page,
      page_size: pageSize
    }
  })
}

export function getMedia(id: string): Promise<ApiMediaItem> {
  return api.get<ApiMediaItem>(`/admin/media/${encodeURIComponent(id)}`)
}

export function updateMedia(
  id: string,
  originalName: string
): Promise<ApiMediaItem> {
  return api.put<ApiMediaItem>(`/admin/media/${encodeURIComponent(id)}`, {
    original_name: originalName
  })
}

export function deleteMedia(id: string): Promise<void> {
  return api.delete<void>(`/admin/media/${encodeURIComponent(id)}`)
}

export function deleteMediaMany(ids: string[]): Promise<void> {
  return api.delete<void>('/admin/media', {
    query: { ids: ids.join(',') }
  })
}

export function uploadMedia(file: File, kind: MediaKind): Promise<ApiMediaItem> {
  const body = new FormData()
  body.append('kind', kind)
  body.append('file', file)
  return api.post<ApiMediaItem>('/admin/media/upload', body, {
    timeoutMs: UPLOAD_TIMEOUT_MS
  })
}
