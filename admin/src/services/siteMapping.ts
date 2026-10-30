/**
 * Go API 数据结构与后台页面 / 导航 / 站点配置模型之间的转换。
 *
 * 后台沿用前端官网的富结构（中英文文案放在同一对象里），
 * 后端站点设置按 key 存 JSON，因此站点配置整体以多语言对象写入。
 */

import type {
  FooterColumn,
  FooterLink,
  Locale,
  NavItem,
  PageBlock,
  PageRecord,
  SiteSettings
} from '~/domain/types'
import { migrateLegacyBlocks } from '@/services/pageService'
import { normalizeSiteSettings, type SiteSettingsInput } from '@/services/settingsService'
import type {
  ApiLinkType,
  ApiLocale,
  ApiMenuDetail,
  ApiMenuItem,
  ApiMenuSaveRequest,
  ApiPageBlock,
  ApiPageDraft,
  ApiPageSummary,
  ApiSeo
} from '@/services/siteApi'

export function toApiLocale(locale: Locale): ApiLocale {
  return locale === 'en' ? 'en' : 'zh-CN'
}

export function fromApiLocale(locale: string): Locale {
  return locale === 'en' ? 'en' : 'zh'
}

/** Unix 秒转成后台显示用的 ISO 时间字符串。 */
export function formatApiTime(seconds: number): string {
  if (!seconds) return ''
  return new Date(seconds * 1000).toISOString()
}

export function pageBlocksToApi(blocks: PageBlock[]): ApiPageBlock[] {
  return blocks as unknown as ApiPageBlock[]
}

function normalizeSeo(seo: ApiSeo | undefined): PageRecord['seo'] {
  return {
    title: seo?.title ?? '',
    description: seo?.description ?? '',
    keywords: Array.isArray(seo?.keywords) ? [...seo.keywords] : []
  }
}

/** 后端页面详情转后台 PageRecord。 */
export function pageFromApi(draft: ApiPageDraft): PageRecord {
  const status = draft.status === 'published' ? 'published' : 'draft'
  const blocks = migrateLegacyBlocks((draft.blocks ?? []) as unknown as PageBlock[])
  const seo = normalizeSeo(draft.seo)

  return {
    id: draft.id,
    groupId: draft.group_id,
    locale: fromApiLocale(draft.locale),
    path: draft.path,
    name: draft.name,
    status,
    updatedAt: formatApiTime(draft.updated_at),
    seo,
    blocks,
    // 后台只用 published.version 展示「线上 vN」，线上区块仍以服务端为准。
    published:
      status === 'published'
        ? {
            version: draft.version,
            blocks: [],
            seo: { ...seo, keywords: [...seo.keywords] },
            publishedAt: formatApiTime(draft.updated_at),
            publishedBy: ''
          }
        : null
  }
}

/**
 * 列表摘要 → 页面记录。
 * 列表接口不返回区块与 SEO，这里只填列表需要的字段；区块数量与编辑内容
 * 由编辑器进入时调 getPage 取详情补充（见 stores/pages 的 ensurePageDetail）。
 */
export function pageSummaryFromApi(summary: ApiPageSummary): PageRecord {
  const status = summary.status === 'published' ? 'published' : 'draft'
  const emptySeo = { title: '', description: '', keywords: [] }

  return {
    id: summary.id,
    groupId: summary.group_id,
    locale: fromApiLocale(summary.locale),
    path: summary.path,
    name: summary.name,
    status,
    updatedAt: formatApiTime(summary.updated_at),
    seo: emptySeo,
    blocks: [],
    published:
      status === 'published'
        ? {
            version: summary.version,
            blocks: [],
            seo: { ...emptySeo, keywords: [] },
            publishedAt: formatApiTime(summary.updated_at),
            publishedBy: summary.updated_by
          }
        : null
  }
}

function labelValue(label: Record<string, string> | undefined, locale: ApiLocale): string {
  if (!label) return ''
  if (locale === 'en') return label.en ?? ''
  return label['zh-CN'] ?? label.zh ?? ''
}

function navItemFromMenu(item: ApiMenuItem, index: number): NavItem {
  return {
    key: item.id || `nav.${index}`,
    path: item.path ?? '',
    zh: labelValue(item.label, 'zh-CN'),
    en: labelValue(item.label, 'en'),
    visible: item.visible !== false
  }
}

function footerLinkFromMenu(item: ApiMenuItem, index: number): FooterLink {
  return {
    key: item.id || `footer.link.${index}`,
    path: item.path ?? '',
    zh: labelValue(item.label, 'zh-CN'),
    en: labelValue(item.label, 'en'),
    visible: item.visible !== false
  }
}

function footerColumnFromMenu(item: ApiMenuItem, index: number): FooterColumn {
  return {
    key: item.id || `footer.column.${index}`,
    title: {
      zh: labelValue(item.label, 'zh-CN'),
      en: labelValue(item.label, 'en')
    },
    visible: item.visible !== false,
    links: (item.children ?? []).map(footerLinkFromMenu)
  }
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null
}

function asString(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined
}

function toLocaleMap(value: unknown): Record<Locale, string> | undefined {
  const record = asRecord(value)
  if (!record) return undefined
  const zh = record['zh-CN'] ?? record.zh
  const en = record.en
  if (typeof zh !== 'string' && typeof en !== 'string') return undefined
  return {
    zh: typeof zh === 'string' ? zh : '',
    en: typeof en === 'string' ? en : ''
  }
}

export interface ApiMenuDetails {
  header?: ApiMenuDetail
  footer?: ApiMenuDetail
  footerBottom?: ApiMenuDetail
}

/** 站点设置 + 三个菜单容器转后台 SiteSettings，缺失字段用默认值补齐。 */
export function siteSettingsFromApi(
  raw: Record<string, unknown>,
  menus: ApiMenuDetails
): SiteSettings {
  const contact = asRecord(raw.contact) ?? {}
  const seo = asRecord(raw.seo) ?? {}
  const footer = asRecord(raw.footer)

  const input: SiteSettingsInput = {
    siteName: toLocaleMap(raw.site_name),
    companyName: toLocaleMap(raw.company_name),
    tagline: toLocaleMap(raw.tagline),
    email: asString(contact.email),
    phone: asString(contact.phone),
    phoneEn: asString(contact.phone_en),
    address: toLocaleMap(contact.address),
    workHours: toLocaleMap(contact.work_hours),
    copyright: toLocaleMap(raw.copyright),
    icp: asString(raw.icp),
    socials: Array.isArray(raw.socials) ? (raw.socials as SiteSettings['socials']) : undefined,
    brand: asRecord(raw.brand) as unknown as SiteSettingsInput['brand'],
    header: asRecord(raw.header) as unknown as SiteSettingsInput['header'],
    footer: footer ? (footer as unknown as SiteSettingsInput['footer']) : undefined,
    allowIndex: typeof raw.allow_index === 'boolean' ? raw.allow_index : undefined,
    analytics: asString(seo.analytics),
    headCode: asString(seo.head_code),
    defaultLocale:
      raw.default_locale === 'en' ? 'en' : raw.default_locale === 'zh' ? 'zh' : undefined
  }

  if (menus.header) {
    input.nav = menus.header.items.map(navItemFromMenu)
  }
  if (menus.footer) {
    input.footer = {
      ...(input.footer ?? {}),
      columns: menus.footer.items.map(footerColumnFromMenu)
    }
  }
  if (menus.footerBottom) {
    input.footer = {
      ...(input.footer ?? {}),
      bottomLinks: menus.footerBottom.items.map(footerLinkFromMenu)
    }
  }

  // 显式的 undefined 覆盖会盖掉 normalizeSiteSettings 的默认值（如 icp / allowIndex），先剔除。
  const cleaned = Object.fromEntries(
    Object.entries(input).filter(([, value]) => value !== undefined)
  ) as SiteSettingsInput
  return normalizeSiteSettings(cleaned)
}

/** 后台 SiteSettings 拆成后端站点设置的 key + value。 */
export function siteSettingsPayload(settings: SiteSettings): Record<string, unknown> {
  return {
    site_name: settings.siteName,
    company_name: settings.companyName,
    tagline: settings.tagline,
    contact: {
      email: settings.email,
      phone: settings.phone,
      phone_en: settings.phoneEn,
      address: settings.address,
      work_hours: settings.workHours
    },
    brand: settings.brand,
    header: settings.header,
    // 栏目和底部链接由菜单表负责，避免两处都能改链接。
    footer: {
      showContact: settings.footer.showContact,
      cta: settings.footer.cta,
      showWordmark: settings.footer.showWordmark,
      wordmark: settings.footer.wordmark
    },
    copyright: settings.copyright,
    icp: settings.icp,
    socials: settings.socials,
    default_locale: settings.defaultLocale,
    allow_index: settings.allowIndex,
    seo: {
      analytics: settings.analytics,
      head_code: settings.headCode
    }
  }
}

function menuLabel(zh: string, en: string): Record<string, string> {
  return { 'zh-CN': zh, en }
}

function linkTypeOf(path: string): ApiLinkType {
  const value = path.trim()
  if (!value) return 'none'
  return /^(https?:)?\/\//i.test(value) || /^(mailto|tel):/i.test(value)
    ? 'external'
    : 'page'
}

/** 顶部导航 → 菜单容器 header。 */
export function headerMenuPayload(settings: SiteSettings): ApiMenuSaveRequest {
  return {
    name: '顶部导航',
    items: settings.nav.map((item, index) => ({
      label: menuLabel(item.zh, item.en),
      path: item.path,
      link_type: linkTypeOf(item.path),
      target: '_self',
      visible: item.visible,
      sort: index
    }))
  }
}

/** 页脚栏目 → 菜单容器 footer（顶级项是栏目，children 是链接）。 */
export function footerMenuPayload(settings: SiteSettings): ApiMenuSaveRequest {
  return {
    name: '底部导航',
    items: settings.footer.columns.map((column, index) => ({
      label: menuLabel(column.title.zh, column.title.en),
      path: '',
      link_type: 'none',
      target: '_self',
      visible: column.visible,
      sort: index,
      children: column.links.map((link, linkIndex) => ({
        label: menuLabel(link.zh, link.en),
        path: link.path,
        link_type: linkTypeOf(link.path),
        target: '_self',
        visible: link.visible,
        sort: linkIndex
      }))
    }))
  }
}

/** 页脚底部链接 → 菜单容器 footer_bottom。 */
export function bottomMenuPayload(settings: SiteSettings): ApiMenuSaveRequest {
  return {
    name: '页脚底部链接',
    items: settings.footer.bottomLinks.map((link, index) => ({
      label: menuLabel(link.zh, link.en),
      path: link.path,
      link_type: linkTypeOf(link.path),
      target: '_self',
      visible: link.visible,
      sort: index
    }))
  }
}
