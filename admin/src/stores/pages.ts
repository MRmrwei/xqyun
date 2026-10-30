import { computed, reactive, ref } from 'vue'
import { defineStore } from 'pinia'
import type { Locale, PageRecord, SiteCollections, SiteSettings } from '~/domain/types'
import type {
  ContactSubmissionRecord,
  LoginSettings,
  PageStatus,
  PersistedDemo,
  SubmissionStatus
} from '@/types/admin'
import { clone, createBlock } from '@/domain/blockSchema'
import { seedLoginSettings, seedSettings } from '@/data/seed/seed'
import { loadDemo, saveDemo } from '@/services/storage'
import { collectionsFor } from '@/services/contentService'
import { normalizeSiteSettings } from '@/services/settingsService'
import {
  duplicatePageRecord,
  formatClock,
  normalizePath,
  pageStatus
} from '@/services/pageService'
import {
  createPage as createPageApi,
  deletePage as deletePageApi,
  getPage,
  getSettings,
  getMenu,
  listMenus,
  listPages,
  publishPage as publishPageApi,
  saveMenu,
  savePageDraft,
  saveSettings as saveSettingsApi,
  translatePage as translatePageApi,
  unpublishPage as unpublishPageApi,
  type ApiMenuDetail,
  type ApiPageSummary
} from '@/services/siteApi'
import {
  bottomMenuPayload,
  footerMenuPayload,
  formatApiTime,
  headerMenuPayload,
  pageBlocksToApi,
  pageFromApi,
  pageSummaryFromApi,
  siteSettingsFromApi,
  siteSettingsPayload,
  toApiLocale
} from '@/services/siteMapping'
import {
  getSubmissionUnreadCount,
  listSubmissions,
  submissionFromApi,
  updateSubmissionStatus,
  type ApiSubmissionQuery,
  type ApiSubmissionStats
} from '@/services/submissionApi'

export interface SubmissionQuery {
  status?: SubmissionStatus | 'all'
  locale?: 'zh' | 'en' | 'all'
  keyword?: string
  page?: number
  pageSize?: number
}

export const usePagesStore = defineStore('pages', () => {
  const persisted = loadDemo()

  const pages = ref<PageRecord[]>([])
  const settings = ref<SiteSettings>(normalizeSiteSettings())
  const submissions = ref<ContactSubmissionRecord[]>([])
  const submissionStats = ref<ApiSubmissionStats>({
    total: 0,
    new: 0,
    processing: 0,
    done: 0,
    spam: 0
  })
  const submissionTotal = ref(0)
  const submissionsLoading = ref(false)
  const submissionsError = ref('')
  const loginSettings = ref<LoginSettings>(
    clone({ ...seedLoginSettings, ...(persisted?.loginSettings ?? {}) })
  )

  // 按需加载：页面列表、站点设置、单页详情各自独立，哪个页面用到哪个才拉。
  const pagesLoading = ref(false)
  const pagesLoaded = ref(false)
  const pagesError = ref('')
  const settingsLoading = ref(false)
  const settingsLoaded = ref(false)
  const settingsError = ref('')
  const detailLoading = ref(false)
  const detailError = ref('')
  /** 已取到完整草稿的页面 id；列表里其余项只有摘要字段。 */
  const detailLoadedIds = reactive(new Set<string>())
  /** 侧边栏「站点编排」的页面总数，只取 total，不依赖列表是否已加载。 */
  const pageCount = ref(0)

  /** 聚合状态：后台外壳与编辑器沿用这两个值。 */
  const loading = computed(
    () => pagesLoading.value || settingsLoading.value || detailLoading.value
  )
  const loadError = computed(() => pagesError.value || settingsError.value)

  const storageError = ref('')
  const lastSavedAt = ref(formatClock())
  let pagesTask: Promise<void> | null = null
  let settingsTask: Promise<void> | null = null
  let detailTask: Promise<PageRecord | null> | null = null
  let detailTaskId = ''
  let badgeTask: Promise<void> | null = null
  let submissionRequestId = 0
  let lastSubmissionQuery: SubmissionQuery = { page: 1, pageSize: 20 }

  /** 页面与站点设置改由 Go API 保存；localStorage 只保留表单留言与登录页设置。 */
  function persist(): void {
    const demo: PersistedDemo = {
      schemaVersion: 1,
      savedAt: new Date().toISOString(),
      loginSettings: clone(loginSettings.value)
    }
    if (saveDemo(demo)) {
      storageError.value = ''
    } else {
      storageError.value = '当前浏览器无法保存设置，刷新后可能丢失。'
    }
  }

  /** 分页取全量页面摘要；列表接口不返回区块，只填列表需要的字段。 */
  async function fetchPageSummaries(): Promise<ApiPageSummary[]> {
    const summaries: ApiPageSummary[] = []
    let page = 1
    for (;;) {
      const result = await listPages({ page, page_size: 100 })
      summaries.push(...result.items)
      pageCount.value = result.total
      if (result.items.length === 0 || summaries.length >= result.total || page >= 50) break
      page += 1
    }
    return summaries
  }

  async function fetchSiteSettings(): Promise<SiteSettings> {
    const [raw, menuList] = await Promise.all([getSettings('zh-CN'), listMenus()])
    const codes = new Set(menuList.items.map((item) => item.code))

    const [header, footer, footerBottom] = await Promise.all([
      codes.has('header') ? getMenu('header') : Promise.resolve(undefined),
      codes.has('footer') ? getMenu('footer') : Promise.resolve(undefined),
      codes.has('footer_bottom') ? getMenu('footer_bottom') : Promise.resolve(undefined)
    ])

    return siteSettingsFromApi(raw as unknown as Record<string, unknown>, {
      header: header as ApiMenuDetail | undefined,
      footer: footer as ApiMenuDetail | undefined,
      footerBottom: footerBottom as ApiMenuDetail | undefined
    })
  }

  /** 加载留言列表与状态统计；只有进入表单留言页才调用。 */
  async function loadSubmissions(query: SubmissionQuery = {}): Promise<void> {
    lastSubmissionQuery = query
    const requestId = ++submissionRequestId
    submissionsLoading.value = true
    submissionsError.value = ''
    try {
      const result = await listSubmissions(toApiSubmissionQuery(query))
      if (requestId !== submissionRequestId) return
      submissions.value = result.items.map(submissionFromApi)
      submissionStats.value = result.stats
      submissionTotal.value = result.total
      // 只有未筛选的全局查询才更新侧边栏红点，避免筛选语言 / 关键词时数量变小。
      if (isGlobalSubmissionQuery(query)) {
        newSubmissionCount.value = result.stats.new
      }
    } catch (error) {
      if (requestId !== submissionRequestId) return
      submissionsError.value =
        error instanceof Error ? error.message : '加载留言失败，请稍后重试'
    } finally {
      if (requestId === submissionRequestId) submissionsLoading.value = false
    }
  }

  /**
   * 只刷新待处理留言数量，供侧边栏红点使用。
   * 走专用计数接口，不拉列表、不算其余状态；不覆盖当前列表与统计卡片。
   * 重复调用会复用同一次请求。
   */
  async function refreshSubmissionBadge(): Promise<void> {
    if (badgeTask) return badgeTask
    badgeTask = (async () => {
      try {
        newSubmissionCount.value = await getSubmissionUnreadCount()
      } catch {
        // 静默失败：保留上一次数量，不打断当前操作。
      }
    })()
    try {
      await badgeTask
    } finally {
      badgeTask = null
    }
  }

  function isGlobalSubmissionQuery(query: SubmissionQuery): boolean {
    return (
      (!query.status || query.status === 'all') &&
      (!query.locale || query.locale === 'all') &&
      !query.keyword?.trim()
    )
  }

  function toApiSubmissionQuery(query: SubmissionQuery): ApiSubmissionQuery {
    const result: ApiSubmissionQuery = {
      page: query.page && query.page > 0 ? query.page : 1,
      page_size: query.pageSize && query.pageSize > 0 ? query.pageSize : 20
    }
    if (query.status && query.status !== 'all') result.status = query.status
    if (query.locale && query.locale !== 'all') {
      result.locale = query.locale === 'en' ? 'en' : 'zh-CN'
    }
    if (query.keyword?.trim()) result.keyword = query.keyword.trim()
    return result
  }

  /** 加载页面列表摘要；已加载过则直接复用。 */
  async function ensurePageList(force = false): Promise<void> {
    if (pagesLoaded.value && !force) return
    if (pagesTask) return pagesTask

    pagesLoading.value = true
    pagesError.value = ''
    pagesTask = (async () => {
      try {
        const summaries = await fetchPageSummaries()
        // 已经取过详情的页面保留完整草稿，避免刷新列表把正在编辑的内容冲掉。
        const details = new Map(
          pages.value
            .filter((page) => detailLoadedIds.has(page.id))
            .map((page) => [page.id, page] as const)
        )
        pages.value = summaries.map(
          (summary) => details.get(summary.id) ?? pageSummaryFromApi(summary)
        )
        pagesLoaded.value = true
        lastSavedAt.value = formatClock()
        // 清掉旧版本地演示缓存；localStorage 只保留登录页设置。
        persist()
      } catch (error) {
        pagesError.value =
          error instanceof Error ? error.message : '加载页面列表失败，请稍后重试'
      } finally {
        pagesLoading.value = false
        pagesTask = null
      }
    })()
    return pagesTask
  }

  /** 加载站点设置与顶部 / 底部导航；已加载过则直接复用。 */
  async function ensureSiteSettings(force = false): Promise<void> {
    if (settingsLoaded.value && !force) return
    if (settingsTask) return settingsTask

    settingsLoading.value = true
    settingsError.value = ''
    settingsTask = (async () => {
      try {
        settings.value = await fetchSiteSettings()
        settingsLoaded.value = true
      } catch (error) {
        settingsError.value =
          error instanceof Error ? error.message : '加载站点配置失败，请稍后重试'
      } finally {
        settingsLoading.value = false
        settingsTask = null
      }
    })()
    return settingsTask
  }

  /** 侧边栏只需要页面总数，不拉列表内容。 */
  async function loadPageCount(): Promise<void> {
    try {
      const result = await listPages({ page: 1, page_size: 1 })
      pageCount.value = result.total
    } catch {
      // 静默失败：侧边栏数字不是关键路径。
    }
  }

  /**
   * 按 id 取单个页面的完整草稿；编辑器、草稿预览与发布前调用。
   * 取到后替换列表里的摘要项，后续读 pages 就是完整记录。
   */
  async function ensurePageDetail(id: string, force = false): Promise<PageRecord | null> {
    if (!id) return null
    if (!force && detailLoadedIds.has(id)) return getById(id) ?? null
    if (detailTask && detailTaskId === id) return detailTask

    detailLoading.value = true
    detailError.value = ''
    detailTaskId = id
    detailTask = (async () => {
      try {
        const record = pageFromApi(await getPage(id))
        const index = pages.value.findIndex((page) => page.id === id)
        if (index >= 0) pages.value[index] = record
        else pages.value.push(record)
        detailLoadedIds.add(id)
        return record
      } catch (error) {
        detailError.value =
          error instanceof Error ? error.message : '加载页面失败，请稍后重试'
        return null
      } finally {
        detailLoading.value = false
        detailTask = null
        detailTaskId = ''
      }
    })()
    return detailTask
  }

  /** 该页面的完整草稿是否已加载（列表项只有摘要时返回 false）。 */
  function hasDetail(id: string): boolean {
    return detailLoadedIds.has(id)
  }

  function getById(id: string): PageRecord | undefined {
    return pages.value.find((page) => page.id === id)
  }

  function getByPath(locale: Locale, path: string): PageRecord | undefined {
    return pages.value.find((page) => page.locale === locale && page.path === path)
  }

  function pairOf(page: PageRecord | null | undefined): PageRecord | undefined {
    if (!page) return undefined
    return pages.value.find(
      (candidate) => candidate.groupId === page.groupId && candidate.locale !== page.locale
    )
  }

  function statusOf(page: PageRecord): PageStatus {
    return pageStatus(page)
  }

  async function createPage(input: {
    name: string
    path: string
    locale: Locale
  }): Promise<PageRecord> {
    const hero = createBlock('hero')
    const draft = await createPageApi({
      locale: toApiLocale(input.locale),
      path: normalizePath(input.path),
      name: input.name,
      seo: { title: input.name, description: '', keywords: [] },
      blocks: pageBlocksToApi(hero ? [hero] : [])
    })

    const page = pageFromApi(draft)
    pages.value.unshift(page)
    // 新建接口返回的就是完整草稿，进编辑器不用再取一次详情。
    detailLoadedIds.add(page.id)
    pageCount.value += 1
    return page
  }

  async function duplicatePage(page: PageRecord): Promise<PageRecord> {
    const template = duplicatePageRecord(page, pages.value)
    const draft = await createPageApi({
      locale: toApiLocale(page.locale),
      path: template.path,
      name: template.name,
      copy_from_id: page.id
    })
    const copy = pageFromApi(draft)
    pages.value.unshift(copy)
    detailLoadedIds.add(copy.id)
    pageCount.value += 1
    return copy
  }

  async function deletePage(pageId: string): Promise<void> {
    await deletePageApi(pageId)
    const index = pages.value.findIndex((page) => page.id === pageId)
    if (index >= 0) pages.value.splice(index, 1)
    detailLoadedIds.delete(pageId)
    pageCount.value = Math.max(0, pageCount.value - 1)
  }

  /** 保存草稿。返回后只同步服务端确定的元数据，避免覆盖请求期间的新编辑。 */
  async function savePage(page: PageRecord): Promise<void> {
    const draft = await savePageDraft(page.id, {
      path: normalizePath(page.path),
      name: page.name,
      seo: page.seo,
      blocks: pageBlocksToApi(page.blocks)
    })

    page.path = draft.path
    page.name = draft.name
    page.status = draft.status === 'published' ? 'published' : 'draft'
    page.updatedAt = formatApiTime(draft.updated_at)
    if (page.status === 'published') {
      page.published = {
        version: draft.version,
        blocks: page.published?.blocks ?? [],
        seo: page.published?.seo ?? clone(page.seo),
        publishedAt: page.published?.publishedAt || formatApiTime(draft.updated_at),
        publishedBy: page.published?.publishedBy ?? ''
      }
    } else {
      page.published = null
    }
  }

  /** 发布 = 用当前草稿覆盖线上内容，不保留历史版本。 */
  async function publishPage(page: PageRecord): Promise<void> {
    const result = await publishPageApi(page.id)
    page.status = 'published'
    page.published = {
      version: result.version,
      blocks: clone(page.blocks),
      seo: clone(page.seo),
      publishedAt: formatApiTime(result.published_at),
      publishedBy: ''
    }
    page.updatedAt = formatApiTime(result.published_at)
  }

  /** 下架：状态回到未发布，并清空线上内容；草稿保留。 */
  async function unpublishPage(page: PageRecord): Promise<void> {
    await unpublishPageApi(page.id)
    page.status = 'draft'
    page.published = null
    page.updatedAt = new Date().toISOString()
  }

  async function createTranslation(page: PageRecord): Promise<PageRecord> {
    const existing = pairOf(page)
    if (existing) return existing

    const targetLocale: Locale = page.locale === 'zh' ? 'en' : 'zh'
    const draft = await translatePageApi(page.id, {
      to_locale: toApiLocale(targetLocale),
      path: page.path
    })
    const translated = pageFromApi(draft)
    pages.value.push(translated)
    detailLoadedIds.add(translated.id)
    pageCount.value += 1
    return translated
  }

  /** 保存站点设置 + 顶部 / 底部导航到 Go API。 */
  async function saveSettings(): Promise<void> {
    // 菜单是整体替换（先删后插），串行提交，避免三个菜单事务并发时互相锁等待。
    await saveSettingsApi('zh-CN', siteSettingsPayload(settings.value))
    await saveMenu('header', headerMenuPayload(settings.value))
    await saveMenu('footer', footerMenuPayload(settings.value))
    await saveMenu('footer_bottom', bottomMenuPayload(settings.value))
    lastSavedAt.value = formatClock()
  }

  /** 保存后台登录页设置（验证码、展示文案，仍为本地演示）。 */
  function saveLoginSettings(next: LoginSettings): void {
    loginSettings.value = clone(next)
    persist()
  }

  /** 只把顶部导航与头部配置恢复为种子数据（仍需手动保存）。 */
  function resetNavigation(): void {
    settings.value.nav = clone(seedSettings.nav)
    settings.value.header = clone(seedSettings.header)
    settings.value.siteName = clone(seedSettings.siteName)
  }

  /** 只把页脚编排（含联系信息、标语、版权、备案号）恢复为种子数据（仍需手动保存）。 */
  function resetFooter(): void {
    settings.value.footer = clone(seedSettings.footer)
    settings.value.tagline = clone(seedSettings.tagline)
    settings.value.copyright = clone(seedSettings.copyright)
    settings.value.icp = seedSettings.icp
    settings.value.email = seedSettings.email
    settings.value.phone = seedSettings.phone
    settings.value.phoneEn = seedSettings.phoneEn
    settings.value.address = clone(seedSettings.address)
  }

  /** 把品牌外观与全站配置恢复为种子数据（仍需手动保存）。 */
  function resetSite(): void {
    settings.value.brand = clone(seedSettings.brand)
    settings.value.defaultLocale = seedSettings.defaultLocale
    settings.value.allowIndex = seedSettings.allowIndex
    settings.value.headCode = seedSettings.headCode
  }

  function collections(locale: Locale): SiteCollections {
    return collectionsFor(locale)
  }

  async function setSubmissionStatus(
    id: string,
    status: SubmissionStatus
  ): Promise<void> {
    const updated = await updateSubmissionStatus(id, status)
    const record = submissionFromApi(updated)
    const index = submissions.value.findIndex((item) => item.id === id)
    if (index >= 0) submissions.value[index] = record
    await loadSubmissions(lastSubmissionQuery)
    // 状态变更可能让待处理数量变化，立即同步红点。
    void refreshSubmissionBadge()
  }

  const submissionCount = computed(() => submissionStats.value.total)
  /** 全站待处理留言数，只由全局查询或定时刷新更新。 */
  const newSubmissionCount = ref(0)

  return {
    pages,
    settings,
    submissions,
    submissionStats,
    submissionTotal,
    submissionsLoading,
    submissionsError,
    loginSettings,
    loading,
    loadError,
    pagesLoading,
    pagesLoaded,
    pagesError,
    settingsLoading,
    settingsLoaded,
    settingsError,
    detailLoading,
    detailError,
    storageError,
    lastSavedAt,
    pageCount,
    submissionCount,
    newSubmissionCount,
    ensurePageList,
    ensureSiteSettings,
    ensurePageDetail,
    hasDetail,
    loadPageCount,
    loadSubmissions,
    refreshSubmissionBadge,
    setSubmissionStatus,
    getById,
    getByPath,
    pairOf,
    statusOf,
    createPage,
    duplicatePage,
    deletePage,
    savePage,
    publishPage,
    unpublishPage,
    createTranslation,
    saveSettings,
    saveLoginSettings,
    resetNavigation,
    resetFooter,
    resetSite,
    collections
  }
})
