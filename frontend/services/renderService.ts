import type {
  BrandSettings,
  FooterColumn,
  FooterCta,
  FooterLink,
  FooterSettings,
  HeaderCta,
  HeaderSettings,
  Locale,
  NavItem,
  PublishedPage,
  SiteCollections,
  SiteSettings,
  SocialLink
} from '~/domain/types'
import type { PageBlock } from '~/domain/blocks'
import { settings as defaultSettings } from '~/data/settings'

/**
 * 官网前台数据边界：只在这里调用 Go 后端。
 * 页面、导航与站点配置一次取回后映射为前端领域类型，组件不感知接口结构。
 */

interface ApiEnvelope<T> {
  code: number
  message: string
  data?: T
}

interface ApiSeo {
  title: string
  description: string
  keywords?: string[]
}

interface ApiPageBlock {
  id: string
  type: string
  props: Record<string, unknown>
}

interface ApiPublishedPage {
  id: string
  group_id: string
  locale: string
  path: string
  name: string
  seo: ApiSeo
  blocks: ApiPageBlock[]
  version: number
  published_at: number
  published_by: string
  alternates?: Record<string, string>
}

interface ApiNavItem {
  id: string
  label: string
  path: string
  link_type?: string
  target?: string
  visible: boolean
  sort?: number
  children?: ApiNavItem[]
}

interface ApiSiteNav {
  header: ApiNavItem[]
  footer: ApiNavItem[]
  footer_bottom: ApiNavItem[]
}

interface ApiRenderPageData {
  page: ApiPublishedPage
  nav: ApiSiteNav
  site: Record<string, unknown>
  collections: Record<string, unknown[]>
}

export interface RenderPageResult {
  page: PublishedPage
  collections: SiteCollections
}

const EMPTY_NAV: ApiSiteNav = { header: [], footer: [], footer_bottom: [] }
const EMPTY_COLLECTIONS: SiteCollections = { news: [], products: [], cases: [] }

function apiBase(): string {
  const config = useRuntimeConfig()
  return import.meta.server ? config.apiBase : config.public.apiBase
}

async function fetchData<T>(
  path: string,
  query: Record<string, string>
): Promise<T | null> {
  try {
    const body = await $fetch<ApiEnvelope<T>>(`${apiBase()}${path}`, {
      query,
      // 标记服务端渲染请求：后端据此跳过成功请求的访问日志，失败仍会记录。
      headers: import.meta.server ? { 'X-Xqyun-Client': 'ssr' } : undefined
    })
    if (!body || body.code !== 0 || body.data === undefined) return null
    return body.data
  } catch {
    // 接口不可用或页面不存在都按「没有数据」处理，由页面层输出 404。
    return null
  }
}

function apiLocale(locale: Locale): string {
  return locale === 'en' ? 'en' : 'zh-CN'
}

function asRecord(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined
}

function asString(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined
}

function asBoolean(value: unknown, fallback: boolean): boolean {
  return typeof value === 'boolean' ? value : fallback
}

function cloneSettings(): SiteSettings {
  return JSON.parse(JSON.stringify(defaultSettings)) as SiteSettings
}

/** 后端多语言字段是 { zh, en } 或 { 'zh-CN', en }，缺失时回退默认值。 */
function localeText(
  value: unknown,
  fallback: Record<Locale, string>
): Record<Locale, string> {
  const record = asRecord(value)
  if (!record) return fallback
  const zh = asString(record.zh) ?? asString(record['zh-CN'])
  const en = asString(record.en)
  return {
    zh: zh ?? fallback.zh,
    en: en ?? fallback.en
  }
}

function mapAlternates(
  raw: Record<string, string> | undefined
): Partial<Record<Locale, string>> | undefined {
  if (!raw) return undefined
  const result: Partial<Record<Locale, string>> = {}
  const zh = raw['zh-CN'] ?? raw.zh
  if (zh) result.zh = zh
  if (raw.en) result.en = raw.en
  return Object.keys(result).length > 0 ? result : undefined
}

function mapPage(raw: ApiPublishedPage): PublishedPage {
  return {
    id: raw.id,
    groupId: raw.group_id,
    locale: raw.locale === 'en' ? 'en' : 'zh',
    path: raw.path,
    name: raw.name,
    seo: {
      title: raw.seo?.title ?? '',
      description: raw.seo?.description ?? '',
      keywords: Array.isArray(raw.seo?.keywords) ? raw.seo.keywords : []
    },
    blocks: (raw.blocks ?? []) as unknown as PageBlock[],
    version: raw.version,
    publishedAt: raw.published_at ? new Date(raw.published_at * 1000).toISOString() : '',
    publishedBy: raw.published_by ?? '',
    alternates: mapAlternates(raw.alternates)
  }
}

function mapNavItems(items: ApiNavItem[]): NavItem[] {
  return items.map((item) => ({
    key: item.id || item.path || item.label,
    path: item.path,
    zh: item.label,
    en: item.label,
    visible: item.visible
  }))
}

function mapFooterColumns(items: ApiNavItem[]): FooterColumn[] {
  return items.map((item) => ({
    key: item.id || item.label,
    title: { zh: item.label, en: item.label },
    visible: item.visible,
    links: (item.children ?? []).map((child) => ({
      key: child.id || child.path || child.label,
      path: child.path,
      zh: child.label,
      en: child.label,
      visible: child.visible
    }))
  }))
}

function mapFooterLinks(items: ApiNavItem[]): FooterLink[] {
  return items.map((item) => ({
    key: item.id || item.path || item.label,
    path: item.path,
    zh: item.label,
    en: item.label,
    visible: item.visible
  }))
}

function mapBrand(raw: Record<string, unknown> | undefined, fallback: BrandSettings): BrandSettings {
  if (!raw) return fallback
  return {
    logo: asString(raw.logo) ?? fallback.logo,
    accent: asString(raw.accent) ?? fallback.accent,
    background: asString(raw.background) ?? fallback.background,
    darkBackground: asString(raw.darkBackground) ?? fallback.darkBackground
  }
}

function mapHeader(
  raw: Record<string, unknown> | undefined,
  fallback: HeaderSettings
): HeaderSettings {
  if (!raw) return fallback
  const announcement = asRecord(raw.announcement)
  return {
    showLanguage: asBoolean(raw.showLanguage, fallback.showLanguage),
    sticky: asBoolean(raw.sticky, fallback.sticky),
    showAnnouncement: asBoolean(raw.showAnnouncement, fallback.showAnnouncement),
    announcement: announcement
      ? {
          text: localeText(announcement.text, fallback.announcement.text),
          highlight: localeText(announcement.highlight, fallback.announcement.highlight),
          linkLabel: localeText(announcement.linkLabel, fallback.announcement.linkLabel),
          linkHref: asString(announcement.linkHref) ?? fallback.announcement.linkHref
        }
      : fallback.announcement,
    ctas: Array.isArray(raw.ctas) ? (raw.ctas as HeaderCta[]) : fallback.ctas
  }
}

function mapFooterCta(value: unknown, fallback: FooterCta): FooterCta {
  const record = asRecord(value)
  if (!record) return fallback
  return {
    label: localeText(record.label, fallback.label),
    path: asString(record.path) ?? fallback.path
  }
}

function mapFooter(
  raw: Record<string, unknown> | undefined,
  nav: ApiSiteNav | null,
  fallback: FooterSettings
): FooterSettings {
  const source = raw ?? {}
  return {
    showContact: asBoolean(source.showContact, fallback.showContact),
    cta: mapFooterCta(source.cta, fallback.cta),
    showWordmark: asBoolean(source.showWordmark, fallback.showWordmark),
    wordmark: asString(source.wordmark) ?? fallback.wordmark,
    columns:
      nav && nav.footer.length > 0 ? mapFooterColumns(nav.footer) : fallback.columns,
    bottomLinks:
      nav && nav.footer_bottom.length > 0
        ? mapFooterLinks(nav.footer_bottom)
        : fallback.bottomLinks
  }
}

/** 后端站点设置 + 三个菜单容器 → 前端 SiteSettings；缺失字段用内置默认值补齐。 */
export function mapSiteSettings(
  raw: Record<string, unknown>,
  nav: ApiSiteNav | null
): SiteSettings {
  const defaults = cloneSettings()
  const contact = asRecord(raw.contact) ?? {}
  const seo = asRecord(raw.seo) ?? {}
  const socials = Array.isArray(raw.socials) ? (raw.socials as SocialLink[]) : undefined

  return {
    ...defaults,
    siteName: localeText(raw.site_name, defaults.siteName),
    companyName: localeText(raw.company_name, defaults.companyName),
    tagline: localeText(raw.tagline, defaults.tagline),
    email: asString(contact.email) ?? defaults.email,
    phone: asString(contact.phone) ?? defaults.phone,
    phoneEn: asString(contact.phone_en) ?? defaults.phoneEn,
    address: localeText(contact.address, defaults.address),
    workHours: localeText(contact.work_hours, defaults.workHours),
    copyright: localeText(raw.copyright, defaults.copyright),
    icp: asString(raw.icp) ?? defaults.icp,
    socials: socials ?? defaults.socials,
    brand: mapBrand(asRecord(raw.brand), defaults.brand),
    header: mapHeader(asRecord(raw.header), defaults.header),
    footer: mapFooter(asRecord(raw.footer), nav, defaults.footer),
    nav: nav && nav.header.length > 0 ? mapNavItems(nav.header) : defaults.nav,
    defaultLocale:
      raw.default_locale === 'en' ? 'en' : raw.default_locale === 'zh' ? 'zh' : defaults.defaultLocale,
    analytics: asString(seo.analytics) ?? defaults.analytics,
    headCode: asString(seo.head_code) ?? defaults.headCode,
    allowIndex: asBoolean(raw.allow_index, defaults.allowIndex)
  }
}

/** 页面渲染数据：页面不存在或未发布时返回 null，前台据此输出 404。 */
export async function getRenderPage(
  locale: Locale,
  path: string
): Promise<RenderPageResult | null> {
  const data = await fetchData<ApiRenderPageData>('/frontend/render/page', {
    path,
    locale: apiLocale(locale)
  })
  if (!data?.page) return null
  return {
    page: mapPage(data.page),
    // 内容库尚未上线，服务端返回空数组，这里保持空列表。
    collections: EMPTY_COLLECTIONS
  }
}

/** 站点级数据（头部 / 页脚），与页面数据分开加载，避免每页重复请求。 */
export async function getSiteSettingsFromApi(locale: Locale): Promise<SiteSettings | null> {
  const [config, nav] = await Promise.all([
    fetchData<Record<string, unknown>>('/frontend/site/config', {
      locale: apiLocale(locale)
    }),
    fetchData<ApiSiteNav>('/frontend/site/nav', { locale: apiLocale(locale) })
  ])
  if (!config && !nav) return null
  return mapSiteSettings(config ?? {}, nav ?? EMPTY_NAV)
}
