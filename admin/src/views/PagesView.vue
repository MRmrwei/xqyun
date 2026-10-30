<template>
  <div v-if="!studioReady" class="adm-page">
    <el-result
      icon="info"
      title="正在加载站点编排"
      sub-title="正在读取页面列表、导航与站点配置…"
    />
  </div>

  <div v-else-if="studioError" class="adm-page">
    <el-result icon="error" title="站点编排加载失败" :sub-title="studioError">
      <template #extra>
        <el-button type="primary" @click="loadStudio(true)">重新加载</el-button>
      </template>
    </el-result>
  </div>

  <div v-else class="adm-page">
    <header class="adm-page-head">
      <div>
        <p class="adm-kicker">SITE STUDIO</p>
        <h1 class="adm-title">站点编排</h1>
        <p class="adm-sub">
          页面内容、顶部 / 底部导航与全站配置统一在这里编排。页面自动保存；导航和站点配置修改后点击「保存设置」。中文和英文分别配置。
        </p>
      </div>
      <div class="adm-actions">
        <el-button v-if="tab === 'pages'" type="primary" @click="openCreate">
          <el-icon><Plus /></el-icon>
          新建页面
        </el-button>
        <span v-else class="studio-hint">修改后需点击「保存设置」</span>
      </div>
    </header>

    <nav class="studio-tabs" aria-label="站点编排模块">
      <button
        v-for="item in tabs"
        :key="item.id"
        class="studio-tab"
        :class="{ active: tab === item.id }"
        :aria-current="tab === item.id ? 'page' : undefined"
        @click="setTab(item.id)"
      >
        <span class="studio-tab-icon"><UiIcon :name="item.icon" :size="16" /></span>
        <span class="studio-tab-text">
          <strong>{{ item.label }}</strong>
          <span>{{ item.note }}</span>
        </span>
        <span class="studio-tab-count">{{ item.count }}</span>
      </button>
    </nav>

    <div v-show="tab === 'pages'">
    <div class="page-stats">
      <div v-for="item in stats" :key="item.label" class="stat-card">
        <span class="stat-label">{{ item.label }}</span>
        <strong class="stat-value">{{ item.value }}</strong>
        <span class="stat-note">{{ item.note }}</span>
      </div>
    </div>

    <section class="adm-card">
      <div class="adm-toolbar">
        <el-input
          v-model="keyword"
          class="adm-search"
          placeholder="搜索页面名称或路径…"
          clearable
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>

        <el-select v-model="localeFilter" style="width: 120px">
          <el-option label="全部语言" value="all" />
          <el-option label="中文" value="zh" />
          <el-option label="English" value="en" />
        </el-select>

        <el-select v-model="statusFilter" style="width: 130px">
          <el-option label="全部状态" value="all" />
          <el-option label="已发布" value="published" />
          <el-option label="未发布" value="draft" />
        </el-select>

        <span class="toolbar-spacer" />
        <span class="mono list-count">{{ filteredPages.length }} / {{ pages.length }} PAGES</span>
      </div>

      <el-table
        v-if="filteredPages.length"
        :data="filteredPages"
        row-key="id"
        @row-dblclick="openEditor"
      >
        <el-table-column label="页面" min-width="200">
          <template #default="{ row }">
            <div class="page-cell">
              <span class="page-icon" :class="{ 'is-en': row.locale === 'en' }">
                <UiIcon :name="row.path === '/' ? 'home' : 'pages'" :size="16" />
              </span>
              <span class="page-name">
                <strong>{{ row.name }}</strong>
              </span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="路径" min-width="200">
          <template #default="{ row }">
            <span class="page-path" :title="fullPath(row)">{{ fullPath(row) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="语言 / 翻译" min-width="150">
          <template #default="{ row }">
            <div class="locale-cell">
              <span class="lang-badge" :class="{ 'is-en': row.locale === 'en' }">
                {{ row.locale === 'en' ? 'EN' : '中' }}
              </span>
              <template v-if="pairOf(row)">
                <span class="pair-text">
                  另有{{ row.locale === 'en' ? '中文' : '英文' }}版
                </span>
              </template>
              <span v-else class="pair-text muted-2">尚未配对</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="发布状态" width="130">
          <template #default="{ row }">
            <el-dropdown
              trigger="click"
              @command="(command: string) => changeStatus(row, command)"
            >
              <span class="status-trigger" title="改变发布状态">
                <StatusTag :status="statusOf(row)" />
                <span class="status-caret">▾</span>
              </span>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="publish">发布 / 重新发布</el-dropdown-item>
                  <el-dropdown-item
                    command="unpublish"
                    :disabled="statusOf(row) === 'draft'"
                  >
                    下架
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>

        <el-table-column label="区块" width="80" align="center">
          <template #default="{ row }">
            <span class="mono">{{ blockCount(row) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="更新时间" width="130">
          <template #default="{ row }">
            <span class="muted-2">{{ formatWhen(row.updatedAt) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="226" align="right">
          <template #default="{ row }">
            <div class="row-actions">
              <button class="icon-action" title="进入编排" @click="openEditor(row)">
                <UiIcon name="edit" :size="15" />
              </button>
              <button class="icon-action" title="预览草稿" @click="openPreview(row)">
                <UiIcon name="eye" :size="15" />
              </button>
              <button
                class="icon-action"
                :title="pairOf(row) ? '前往翻译版本' : '创建翻译版本'"
                @click="openTranslation(row)"
              >
                <UiIcon name="globe2" :size="15" />
              </button>
              <button class="icon-action" title="复制页面" @click="duplicate(row)">
                <UiIcon name="copy" :size="15" />
              </button>
              <button class="icon-action danger" title="删除页面" @click="remove(row)">
                <UiIcon name="trash" :size="15" />
              </button>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <el-empty
        v-else
        :description="pages.length ? '没有符合条件的页面，换个筛选条件试试' : '还没有页面，先新建一个'"
      >
        <el-button type="primary" @click="openCreate">新建页面</el-button>
      </el-empty>
    </section>

    <!-- 新建页面 -->
    <el-dialog v-model="createOpen" title="新建页面" width="520px">
      <div class="create-form">
        <div class="prop-field">
          <span class="prop-label">页面名称 <span class="req">*</span></span>
          <el-input v-model="createForm.name" placeholder="例如：关于我们" />
        </div>

        <div class="prop-field">
          <span class="prop-label">语言</span>
          <el-radio-group v-model="createForm.locale">
            <el-radio-button value="zh">中文（无前缀）</el-radio-button>
            <el-radio-button value="en">English（/en）</el-radio-button>
          </el-radio-group>
        </div>

        <div class="prop-field">
          <span class="prop-label">访问路径</span>
          <div class="path-input">
            <span class="mono">{{ createForm.locale === 'en' ? '/en' : '' }}</span>
            <el-input v-model="createForm.path" placeholder="/about" />
          </div>
          <span class="prop-help">最终地址：{{ createPreviewPath }}</span>
        </div>

        <p v-if="createIssue" class="create-issue">{{ createIssue }}</p>
      </div>

      <template #footer>
        <el-button @click="createOpen = false">取消</el-button>
        <el-button type="primary" :disabled="Boolean(createIssue)" @click="submitCreate">
          创建并编排
        </el-button>
      </template>
    </el-dialog>

    <!-- 发布校验 -->
    <PublishDialog
      :open="publishOpen"
      :page="publishTarget"
      :checks="publishChecks"
      :ready="publishReady"
      @close="publishOpen = false"
      @confirm="confirmPublish"
    />

    <!-- 草稿预览 -->
    <el-dialog
      v-model="previewOpen"
      class="preview-dialog"
      title="草稿预览"
      width="min(1180px, 92vw)"
      destroy-on-close
    >
      <PreviewFrame
        v-if="previewTarget"
        :page="previewTarget"
        :locale="previewTarget.locale"
        :collections="previewCollections"
        :settings="pagesStore.settings"
      />
    </el-dialog>
    </div>

    <HeaderNavPanel v-if="tab === 'header'" />
    <FooterNavPanel v-if="tab === 'footer'" />
    <SiteSettingsPanel v-if="tab === 'settings'" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search } from '@element-plus/icons-vue'
import type { Locale, PageRecord } from '~/domain/types'
import type { PageStatus } from '@/types/admin'
import { usePagesStore } from '@/stores/pages'
import { isPathTaken, isReservedPath, normalizePath } from '@/services/pageService'
import { usePublishValidation } from '@/composables/usePublishValidation'
import StatusTag from '@/components/common/StatusTag.vue'
import PreviewFrame from '@/components/preview/PreviewFrame.vue'
import PublishDialog from '@/components/editor/PublishDialog.vue'
import HeaderNavPanel from '@/components/navigation/HeaderNavPanel.vue'
import FooterNavPanel from '@/components/navigation/FooterNavPanel.vue'
import SiteSettingsPanel from '@/components/settings/SiteSettingsPanel.vue'
import UiIcon from '~/components/UiIcon.vue'

type StudioTab = 'pages' | 'header' | 'footer' | 'settings'

const route = useRoute()
const router = useRouter()
const pagesStore = usePagesStore()
const { pages } = storeToRefs(pagesStore)

const tab = computed<StudioTab>(() => {
  const view = route.query.view
  return view === 'header' || view === 'footer' || view === 'settings' ? view : 'pages'
})

/* 站点编排页按需加载：页面摘要 + 站点设置（四个 Tab 与预览都要用）。 */
// 从编辑器返回时数据已在内存里，直接用，避免闪一下加载态。
const studioReady = ref(pagesStore.pagesLoaded && pagesStore.settingsLoaded)
const studioError = ref('')

async function loadStudio(force = false) {
  studioError.value = ''
  await Promise.all([
    pagesStore.ensurePageList(force),
    pagesStore.ensureSiteSettings(force)
  ])
  studioError.value = pagesStore.pagesError || pagesStore.settingsError
  studioReady.value = true
}

onMounted(() => loadStudio())

const tabs = computed(() => [
  {
    id: 'pages' as StudioTab,
    label: '页面内容',
    note: '区块编排、预览与发布',
    icon: 'pages',
    count: pages.value.length
  },
  {
    id: 'header' as StudioTab,
    label: '顶部导航',
    note: '菜单项、按钮与吸顶',
    icon: 'menu',
    count: pagesStore.settings.nav.filter((item) => item.visible).length
  },
  {
    id: 'footer' as StudioTab,
    label: '底部导航',
    note: '页脚栏目、链接与版权',
    icon: 'layout',
    count: pagesStore.settings.footer.columns.filter((column) => column.visible).length
  },
  {
    id: 'settings' as StudioTab,
    label: '站点配置',
    note: 'Logo、主色与收录',
    icon: 'settings',
    count: 3
  }
])

function setTab(next: StudioTab) {
  if (next === tab.value) return
  const query = { ...route.query }
  if (next === 'pages') delete query.view
  else query.view = next
  router.replace({ name: 'pages', query })
}

const keyword = ref('')
const localeFilter = ref<'all' | Locale>('all')
const statusFilter = ref<'all' | PageStatus>('all')

const filteredPages = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  return pages.value
    .filter((page) => {
      if (localeFilter.value !== 'all' && page.locale !== localeFilter.value) return false
      if (statusFilter.value !== 'all' && pagesStore.statusOf(page) !== statusFilter.value) {
        return false
      }
      if (!text) return true
      return `${page.name} ${page.path}`.toLowerCase().includes(text)
    })
    .slice()
    .sort((a, b) => {
      if (a.locale !== b.locale) return a.locale === 'zh' ? -1 : 1
      if (a.path === '/') return -1
      if (b.path === '/') return 1
      return a.path.localeCompare(b.path)
    })
})

const stats = computed(() => {
  const all = pages.value
  return [
    { label: '全部页面', value: all.length, note: '中英文合计' },
    {
      label: '已发布',
      value: all.filter((page) => pagesStore.statusOf(page) === 'published').length,
      note: '与线上一致'
    },
    {
      label: '待发布',
      value: all.filter((page) => pagesStore.statusOf(page) !== 'published').length,
      note: '草稿'
    },
    {
      label: '英文页面',
      value: all.filter((page) => page.locale === 'en').length,
      note: '含未发布'
    }
  ]
})

function asPage(row: unknown): PageRecord {
  return row as PageRecord
}

function statusOf(row: unknown): PageStatus {
  return pagesStore.statusOf(asPage(row))
}

function pairOf(row: unknown): PageRecord | undefined {
  return pagesStore.pairOf(asPage(row))
}

/** 区块数要打开过编辑器才有；列表只拿摘要时显示占位。 */
function blockCount(row: unknown): string | number {
  const page = asPage(row)
  return pagesStore.hasDetail(page.id) ? page.blocks.length : '—'
}

function fullPath(row: unknown): string {
  const page = asPage(row)
  const prefix = page.locale === 'en' ? '/en' : ''
  return page.path === '/' ? prefix || '/' : `${prefix}${page.path}`
}

function formatWhen(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value || '—'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date)
}

function openEditor(row: unknown) {
  router.push({ name: 'page-editor', params: { id: asPage(row).id } })
}

/* 新建 */
const createOpen = ref(false)
const createForm = reactive<{
  name: string
  path: string
  locale: Locale
}>({
  name: '',
  path: '',
  locale: 'zh'
})

const createPath = computed(() => normalizePath(createForm.path))
const createPreviewPath = computed(() => {
  const prefix = createForm.locale === 'en' ? '/en' : ''
  return createPath.value === '/' ? prefix || '/' : `${prefix}${createPath.value}`
})

const createIssue = computed(() => {
  if (!createForm.name.trim()) return '请填写页面名称。'
  if (isReservedPath(createPath.value)) {
    return `「${createPath.value}」属于系统保留路径，不能用于页面。`
  }
  if (isPathTaken(pages.value, createForm.locale, createPath.value)) {
    return '该路径在当前语言下已被占用。'
  }
  return ''
})

function openCreate() {
  createForm.name = ''
  createForm.path = ''
  createForm.locale = 'zh'
  createOpen.value = true
}

async function submitCreate() {
  if (createIssue.value) return
  try {
    const page = await pagesStore.createPage({
      name: createForm.name.trim(),
      path: createPath.value,
      locale: createForm.locale
    })
    createOpen.value = false
    ElMessage.success(`已创建「${page.name}」`)
    openEditor(page)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '创建页面失败')
  }
}

/* 复制 / 翻译 / 删除 */
async function duplicate(row: unknown) {
  try {
    const copy = await pagesStore.duplicatePage(asPage(row))
    ElMessage.success(`已复制为「${copy.name}」`)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '复制页面失败')
  }
}

async function openTranslation(row: unknown) {
  const page = asPage(row)
  const existing = pagesStore.pairOf(page)
  try {
    const target = existing ?? (await pagesStore.createTranslation(page))
    ElMessage.success(existing ? '已存在配对版本，正在打开' : '已创建翻译草稿')
    openEditor(target)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '创建翻译版本失败')
  }
}

async function remove(row: unknown) {
  const page = asPage(row)
  try {
    await ElMessageBox.confirm(
      `确定删除「${page.name}」吗？该页面的草稿会一起删除，且不可恢复。`,
      '删除页面',
      { type: 'warning', confirmButtonText: '确认删除', cancelButtonText: '取消' }
    )
  } catch {
    return
  }

  try {
    await pagesStore.deletePage(page.id)
    ElMessage.success('页面已删除')
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '删除页面失败')
  }
}

/* 发布 */
const publishOpen = ref(false)
const publishTarget = ref<PageRecord | null>(null)
const publishPair = computed(() => pagesStore.pairOf(publishTarget.value))

const { checks: publishChecks, ready: publishReady } = usePublishValidation(
  publishTarget,
  pages,
  publishPair
)

async function openPublish(row: unknown) {
  const page = asPage(row)
  const detail = await pagesStore.ensurePageDetail(page.id)
  if (!detail) {
    ElMessage.error(pagesStore.detailError || '加载页面内容失败，请稍后重试')
    return
  }
  publishTarget.value = detail
  publishOpen.value = true
}

async function confirmPublish() {
  const page = publishTarget.value
  if (!page) return
  try {
    await pagesStore.publishPage(page)
    publishOpen.value = false
    ElMessage.success(`「${page.name}」已发布`)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '发布失败，请稍后重试')
  }
}

function changeStatus(row: unknown, command: string) {
  if (command === 'publish') {
    openPublish(row)
    return
  }
  if (command === 'unpublish') {
    unpublish(row)
  }
}

async function unpublish(row: unknown) {
  const page = asPage(row)
  try {
    await ElMessageBox.confirm(
      `确定下架「${page.name}」吗？下架后前台不再展示，草稿会保留。`,
      '下架页面',
      { type: 'warning', confirmButtonText: '确认下架', cancelButtonText: '取消' }
    )
  } catch {
    return
  }

  try {
    await pagesStore.unpublishPage(page)
    ElMessage.success(`「${page.name}」已下架`)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '下架失败，请稍后重试')
  }
}

/* 预览 */
const previewOpen = ref(false)
const previewTarget = ref<PageRecord | null>(null)
const previewCollections = computed(() =>
  pagesStore.collections(previewTarget.value?.locale ?? 'zh')
)

async function openPreview(row: unknown) {
  const page = asPage(row)
  const detail = await pagesStore.ensurePageDetail(page.id)
  if (!detail) {
    ElMessage.error(pagesStore.detailError || '加载页面内容失败，请稍后重试')
    return
  }
  previewTarget.value = detail
  previewOpen.value = true
}
</script>

<style scoped>
.studio-tabs {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 6px;
  padding: 5px;
  margin-bottom: 16px;
  background: var(--adm-surface);
  border: 1px solid var(--adm-line);
  border-radius: 12px;
}

.studio-tab {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 10px 12px;
  border: 1px solid transparent;
  border-radius: 9px;
  background: transparent;
  text-align: left;
  cursor: pointer;
  transition: background 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease;
}

.studio-tab:hover {
  background: rgba(255, 255, 255, 0.6);
}

.studio-tab.active {
  background: #fff;
  border-color: var(--adm-line);
  box-shadow: 0 1px 2px rgba(20, 21, 24, 0.04);
}

.studio-tab-icon {
  width: 30px;
  height: 30px;
  flex: none;
  display: grid;
  place-items: center;
  border-radius: 8px;
  background: #f1ede5;
  color: var(--adm-muted);
}

.studio-tab.active .studio-tab-icon {
  background: var(--adm-accent-soft);
  color: var(--adm-accent);
}

.studio-tab-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.studio-tab-text strong {
  font-size: 12px;
  font-weight: 600;
  color: var(--adm-ink);
}

.studio-tab-text span {
  font-size: 9.5px;
  color: var(--adm-muted-2);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.studio-tab-count {
  margin-left: auto;
  min-width: 22px;
  height: 20px;
  padding: 0 6px;
  display: grid;
  place-items: center;
  border-radius: 999px;
  background: #f1ede5;
  color: var(--adm-muted);
  font-family: var(--adm-mono);
  font-size: 9px;
}

.studio-tab.active .studio-tab-count {
  background: var(--adm-ink);
  color: #f6f2ea;
}

.studio-hint {
  font-size: 10px;
  color: var(--adm-muted-2);
}

.page-stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 16px;
}

.stat-card {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 14px 16px;
  background: var(--adm-card);
  border: 1px solid var(--adm-line);
  border-radius: 11px;
  box-shadow: var(--adm-shadow);
}

.stat-label {
  font-family: var(--adm-mono);
  font-size: 8.5px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--adm-muted-2);
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  letter-spacing: -0.04em;
  line-height: 1.15;
}

.stat-note {
  font-size: 10px;
  color: var(--adm-muted-2);
}

.list-count {
  font-size: 9px;
  color: var(--adm-muted-2);
}

.locale-cell {
  display: flex;
  align-items: center;
  gap: 7px;
  flex-wrap: wrap;
}

.pair-text {
  font-size: 10.5px;
  color: var(--adm-muted);
}

.create-form {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.create-form .prop-label {
  display: flex;
  align-items: center;
  gap: 4px;
}

.create-form .req {
  color: var(--adm-accent);
}

.create-form .prop-help {
  font-size: 10px;
  color: var(--adm-muted-2);
}

.path-input {
  display: flex;
  align-items: center;
  border: 1px solid var(--adm-line);
  border-radius: 8px;
  overflow: hidden;
  background: #fff;
}

.path-input > span {
  padding-left: 10px;
  font-size: 10px;
  color: var(--adm-muted-2);
  white-space: nowrap;
}

.path-input :deep(.el-input__wrapper) {
  box-shadow: none !important;
  background: transparent;
}

.create-issue {
  margin: 0;
  padding: 9px 11px;
  border-radius: 7px;
  background: var(--adm-red-soft);
  color: #b23a25;
  font-size: 10.5px;
  line-height: 1.55;
}

.status-trigger {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  cursor: pointer;
  user-select: none;
}

.status-caret {
  color: var(--adm-muted-2);
  font-size: 9px;
  line-height: 1;
}

@media (max-width: 1100px) {
  .studio-tabs {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .page-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 760px) {
  .studio-tabs {
    grid-template-columns: 1fr;
  }
  .studio-tab-text span {
    white-space: normal;
  }
}
</style>
