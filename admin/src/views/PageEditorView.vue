<template>
  <div v-if="page" class="editor-shell">
    <div class="editor-toolbar">
      <button class="editor-back" title="返回页面列表" @click="goBack">
        <UiIcon name="arrow-left" :size="15" />
      </button>

      <div class="editor-title">
        <div class="editor-title-row">
          <h1>{{ page.name }}</h1>
          <StatusTag :status="currentStatus" />
        </div>
        <div class="editor-sub">
          <span class="mono">{{ fullPath }}</span>
          <span class="dot" />
          <span>{{ formatWhen(page.updatedAt) }}</span>
          <span class="dot" />
          <span>{{ page.blocks.length }} 个区块</span>
        </div>
      </div>

      <div class="editor-locales">
        <button
          class="locale-tab"
          :class="{ active: page.locale === 'zh', missing: dotFor('zh') === 'missing' }"
          @click="switchLocale('zh')"
        >
          <span class="locale-dot" :class="dotFor('zh')" />
          中文
        </button>
        <button
          class="locale-tab"
          :class="{ active: page.locale === 'en', missing: dotFor('en') === 'missing' }"
          @click="switchLocale('en')"
        >
          <span class="locale-dot" :class="dotFor('en')" />
          EN
          <span v-if="dotFor('en') === 'missing'" class="locale-hint">未创建</span>
        </button>
      </div>

      <div class="editor-actions">
        <span class="save-state" :class="saveState">{{ saveLabel }}</span>

        <div class="history-btns">
          <button title="撤销" :disabled="!canUndo" @click="undo">
            <UiIcon name="undo" :size="13" />
          </button>
          <button title="重做" :disabled="!canRedo" @click="redo">
            <UiIcon name="redo" :size="13" />
          </button>
        </div>

        <el-button size="small" :loading="savingDraft" @click="saveDraft">
          保存草稿
        </el-button>
        <el-button size="small" @click="previewOpen = true">预览</el-button>
        <el-button
          v-if="currentStatus === 'published'"
          size="small"
          @click="unpublishCurrent"
        >
          下架
        </el-button>
        <el-button size="small" type="primary" @click="openPublish">发布</el-button>
      </div>
    </div>

    <div class="editor-body">
      <BlockStructure
        :page="page"
        :selected-id="selectedId"
        :mode="mode"
        @select="selectBlock"
        @add="libraryOpen = true"
        @duplicate="duplicateBlock"
        @remove="removeBlock"
        @reorder="reorderBlocks"
        @page-settings="mode = 'page'"
      />

      <section class="editor-panel editor-panel--props">
        <div class="panel-props-head">
          <div>
            <p class="adm-kicker">{{ mode === 'page' ? 'PAGE SETTINGS' : 'BLOCK PROPS' }}</p>
            <h2>{{ propsTitle }}</h2>
          </div>
          <span class="type-chip">{{ mode === 'page' ? 'SEO' : selectedBlock?.type || 'EMPTY' }}</span>
        </div>

        <div class="props-scroll">
          <PageSettingsForm
            v-if="mode === 'page'"
            :page="page"
            :pair="pair"
            :pair-status="pairStatus ?? 'missing'"
            :status="currentStatus"
            :locale="page.locale"
            @update-page="updatePageMeta"
            @update-seo="updateSeo"
            @update-keywords="updateKeywords"
            @switch-locale="switchToPairLocale"
          />

          <SchemaForm
            v-else-if="selectedBlock && selectedSchema"
            :schema="selectedSchema"
            :model="selectedProps"
            @update="updateProp"
          />

          <div v-else class="props-empty">
            <UiIcon name="blocks" :size="28" />
            <p>从左侧选择一个区块，或点「添加区块」开始编排这个页面。</p>
          </div>
        </div>
      </section>

      <div class="editor-panel editor-panel--preview">
        <PreviewFrame
          :page="page"
          :locale="page.locale"
          :collections="collections"
          :settings="pagesStore.settings"
          :focus-request="focusRequest"
        />
      </div>
    </div>

    <BlockLibraryDialog
      :open="libraryOpen"
      @close="libraryOpen = false"
      @add="addBlock"
    />

    <PublishDialog
      :open="publishOpen"
      :page="page"
      :checks="publishChecks"
      :ready="publishReady"
      @close="publishOpen = false"
      @confirm="confirmPublish"
    />

    <el-dialog
      v-model="previewOpen"
      class="preview-dialog"
      title="草稿预览"
      width="min(1180px, 92vw)"
      destroy-on-close
    >
      <PreviewFrame
        :page="page"
        :locale="page.locale"
        :collections="collections"
        :settings="pagesStore.settings"
        :focus-request="focusRequest"
      />
    </el-dialog>
  </div>

  <div v-else-if="loadFailure" class="adm-page">
    <el-result icon="error" title="页面加载失败" :sub-title="loadFailure">
      <template #extra>
        <el-button @click="router.push('/pages')">返回页面列表</el-button>
        <el-button type="primary" :loading="pagesStore.loading" @click="reload">
          重新加载
        </el-button>
      </template>
    </el-result>
  </div>

  <div v-else-if="pagesStore.loading" class="adm-page">
    <el-result icon="info" title="正在加载页面" sub-title="正在读取页面数据…" />
  </div>

  <div v-else class="adm-page">
    <el-result
      icon="warning"
      title="页面不存在"
      sub-title="这个页面可能已经被删除，或者链接里的 ID 不正确。"
    >
      <template #extra>
        <el-button type="primary" @click="router.push('/pages')">返回页面列表</el-button>
      </template>
    </el-result>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { Locale, PageBlock, PageRecord } from '~/domain/types'
import { blockSchemas, clone, createBlock, uid } from '@/domain/blockSchema'
import { usePagesStore } from '@/stores/pages'
import { useAutosave } from '@/composables/useAutosave'
import { useEditorHistory } from '@/composables/useEditorHistory'
import { usePublishValidation } from '@/composables/usePublishValidation'
import StatusTag from '@/components/common/StatusTag.vue'
import BlockStructure from '@/components/editor/BlockStructure.vue'
import BlockLibraryDialog from '@/components/editor/BlockLibraryDialog.vue'
import PageSettingsForm from '@/components/editor/PageSettingsForm.vue'
import PublishDialog from '@/components/editor/PublishDialog.vue'
import SchemaForm from '@/components/form/SchemaForm.vue'
import PreviewFrame from '@/components/preview/PreviewFrame.vue'
import UiIcon from '~/components/UiIcon.vue'

const route = useRoute()
const router = useRouter()
const pagesStore = usePagesStore()
const { pages } = storeToRefs(pagesStore)

const pageId = computed(() => {
  return typeof route.params.id === 'string' ? route.params.id : ''
})

/**
 * 编辑器需要三份数据：页面列表（翻译配对与路径校验）、站点设置（预览头部页脚）
 * 与当前页的完整草稿。三者都到齐才渲染，避免用摘要（区块为空）进入编辑。
 */
const ready = computed(
  () =>
    pagesStore.pagesLoaded &&
    pagesStore.settingsLoaded &&
    pagesStore.hasDetail(pageId.value)
)

const page = computed<PageRecord | null>(() => {
  if (!ready.value) return null
  return (pageId.value && pagesStore.getById(pageId.value)) || null
})

const loadFailure = computed(
  () => pagesStore.detailError || pagesStore.pagesError || pagesStore.settingsError
)

/* 进入即按 id 取详情；换页面时重新取。 */
void pagesStore.ensurePageList()
void pagesStore.ensureSiteSettings()
void pagesStore.ensurePageDetail(pageId.value)
watch(pageId, (id) => {
  if (id) void pagesStore.ensurePageDetail(id)
})

const currentStatus = computed(() => (page.value ? pagesStore.statusOf(page.value) : 'draft'))
const pair = computed(() => pagesStore.pairOf(page.value))
const pairStatus = computed(() => (pair.value ? pagesStore.statusOf(pair.value) : undefined))
const collections = computed(() => pagesStore.collections(page.value?.locale ?? 'zh'))

const fullPath = computed(() => {
  if (!page.value) return ''
  const prefix = page.value.locale === 'en' ? '/en' : ''
  return page.value.path === '/' ? prefix || '/' : `${prefix}${page.value.path}`
})

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

/* 编辑状态 */
const selectedId = ref<string | null>(null)
const mode = ref<'block' | 'page'>('block')
/** 每次点击区块递增 token，预览据此滚动到对应区块。 */
const focusRequest = ref<{ id: string; token: number } | null>(null)
let focusToken = 0

function focusPreviewBlock(id: string) {
  focusToken += 1
  focusRequest.value = { id, token: focusToken }
}
const libraryOpen = ref(false)
const previewOpen = ref(false)
const publishOpen = ref(false)
const savingDraft = ref(false)

const selectedBlock = computed<PageBlock | null>(() => {
  if (!page.value || !selectedId.value) return null
  return page.value.blocks.find((block) => block.id === selectedId.value) ?? null
})

const selectedSchema = computed(() =>
  selectedBlock.value ? blockSchemas[selectedBlock.value.type] : null
)

const selectedProps = computed(
  () => (selectedBlock.value?.props ?? {}) as Record<string, unknown>
)

const propsTitle = computed(() => {
  if (mode.value === 'page') return '页面设置与 SEO'
  if (selectedBlock.value) return blockSchemas[selectedBlock.value.type]?.label ?? '区块属性'
  return '区块属性'
})

/* 自动保存与撤销 / 重做 */
// savePage 会更新 updatedAt，这个写入不应再被视为一次编辑。
const autosave = useAutosave(
  () => page.value,
  (target) => muteWhile(() => pagesStore.savePage(target)),
  3000,
  (error) => {
    ElMessage.error(error instanceof Error ? error.message : '自动保存失败，请稍后重试')
  }
)
const saveState = autosave.state
const saveLabel = autosave.label
const history = useEditorHistory(page)
const canUndo = history.canUndo
const canRedo = history.canRedo

/** 手动保存草稿，不发布；自动保存仍在停止编辑 3 秒后触发。 */
async function saveDraft() {
  if (!page.value || savingDraft.value) return
  savingDraft.value = true
  try {
    const saved = await autosave.saveNow()
    if (saved) ElMessage.success('草稿已保存')
  } finally {
    savingDraft.value = false
  }
}

function handleSaveShortcut(event: KeyboardEvent) {
  if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's') {
    event.preventDefault()
    void saveDraft()
  }
}

onMounted(() => window.addEventListener('keydown', handleSaveShortcut))
onBeforeUnmount(() => window.removeEventListener('keydown', handleSaveShortcut))

let activePageId = ''
let muteWatch = false

watch(
  page,
  (current) => {
    if (!current) return
    if (current.id !== activePageId) {
      activePageId = current.id
      history.reset()
      autosave.reset()
      selectedId.value = current.blocks[0]?.id ?? null
      mode.value = 'block'
      return
    }
    if (muteWatch) return
    history.record()
    autosave.markDirty()
  },
  { deep: true, immediate: true }
)

/** 保存 / 发布接口返回后会更新页面字段，期间暂停「有未保存修改」的监听。 */
async function muteWhile(action: () => Promise<void>) {
  muteWatch = true
  try {
    await action()
  } finally {
    await nextTick()
    muteWatch = false
  }
}

async function reload() {
  void pagesStore.ensurePageList(true)
  void pagesStore.ensureSiteSettings(true)
  await pagesStore.ensurePageDetail(pageId.value, true)
  if (pagesStore.detailError) {
    ElMessage.error(pagesStore.detailError)
  }
}

function undo() {
  if (history.undo()) autosave.markDirty()
}

function redo() {
  if (history.redo()) autosave.markDirty()
}

/* 区块操作 */
function selectBlock(block: PageBlock) {
  selectedId.value = block.id
  mode.value = 'block'
  focusPreviewBlock(block.id)
}

function addBlock(type: PageBlock['type']) {
  if (!page.value) return
  const block = createBlock(type)
  if (!block) return
  page.value.blocks.push(block)
  selectedId.value = block.id
  mode.value = 'block'
  focusPreviewBlock(block.id)
  libraryOpen.value = false
  history.record(true)
}

function duplicateBlock(block: PageBlock) {
  if (!page.value) return
  const index = page.value.blocks.findIndex((item) => item.id === block.id)
  if (index < 0) return
  const copy = { ...clone(block), id: uid('blk') } as PageBlock
  page.value.blocks.splice(index + 1, 0, copy)
  selectedId.value = copy.id
  mode.value = 'block'
  history.record(true)
}

async function removeBlock(block: PageBlock) {
  if (!page.value) return
  try {
    await ElMessageBox.confirm(
      `确定删除「${blockSchemas[block.type]?.label || block.type}」区块吗？`,
      '删除区块',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
    )
    const index = page.value.blocks.findIndex((item) => item.id === block.id)
    if (index < 0) return
    page.value.blocks.splice(index, 1)
    if (selectedId.value === block.id) {
      selectedId.value = page.value.blocks[index]?.id ?? page.value.blocks[index - 1]?.id ?? null
    }
    history.record(true)
  } catch {
    // cancelled
  }
}

function reorderBlocks(blocks: PageBlock[]) {
  if (!page.value) return
  page.value.blocks = blocks
  history.record(true)
}

function updateProp(key: string, value: unknown) {
  if (!selectedBlock.value) return
  ;(selectedBlock.value.props as unknown as Record<string, unknown>)[key] = value
}

/* 页面设置 */
function updatePageMeta(key: 'name' | 'path', value: string) {
  if (!page.value) return
  page.value[key] = value
}

function updateSeo(key: 'title' | 'description', value: string) {
  if (!page.value) return
  page.value.seo[key] = value
}

function updateKeywords(value: string) {
  if (!page.value) return
  page.value.seo.keywords = value
    .split(/[,，]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

/* 语言切换 */
function dotFor(locale: Locale): string {
  if (!page.value) return 'neutral'
  if (page.value.locale === locale) {
    return currentStatus.value === 'published' ? 'success' : 'neutral'
  }
  if (!pair.value) return 'missing'
  return pairStatus.value === 'published' ? 'success' : 'neutral'
}

async function switchLocale(locale: Locale) {
  const current = page.value
  if (!current || current.locale === locale) return
  await autosave.flush()
  const existing = pagesStore.pairOf(current)
  try {
    const target = existing ?? (await pagesStore.createTranslation(current))
    if (!existing) ElMessage.success('已创建翻译草稿，文本需要重新填写')
    await router.push({ name: 'page-editor', params: { id: target.id } })
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '创建翻译版本失败')
  }
}

function switchToPairLocale() {
  switchLocale(page.value?.locale === 'zh' ? 'en' : 'zh')
}

/* 发布 */
const { checks: publishChecks, ready: publishReady } = usePublishValidation(
  page,
  pages,
  pair
)

async function openPublish() {
  await autosave.flush()
  publishOpen.value = true
}

async function confirmPublish() {
  const current = page.value
  if (!current) return
  await autosave.flush()
  try {
    await muteWhile(() => pagesStore.publishPage(current))
    publishOpen.value = false
    ElMessage.success(`「${current.name}」已发布`)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '发布失败，请稍后重试')
  }
}

async function unpublishCurrent() {
  const current = page.value
  if (!current) return
  try {
    await ElMessageBox.confirm(
      `确定下架「${current.name}」吗？下架后前台不再展示，草稿会保留。`,
      '下架页面',
      { type: 'warning', confirmButtonText: '确认下架', cancelButtonText: '取消' }
    )
  } catch {
    return
  }

  try {
    await muteWhile(() => pagesStore.unpublishPage(current))
    ElMessage.success(`「${current.name}」已下架`)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '下架失败，请稍后重试')
  }
}

/* 离开编辑器 */
async function goBack() {
  await autosave.flush()
  await router.push('/pages')
}
</script>
