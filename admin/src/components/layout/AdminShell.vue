<template>
  <div class="admin-shell">
    <aside class="admin-side">
      <p class="side-label">工作台</p>
      <nav class="side-nav">
        <router-link to="/pages" class="side-link" :class="{ active: isPagesActive }">
          <el-icon><Document /></el-icon>
          <span>站点编排</span>
          <span class="side-count">{{ pagesStore.pageCount }}</span>
        </router-link>
      </nav>
      <p class="side-note">页面内容 · 顶部导航 · 底部导航 · 站点配置</p>

      <p class="side-label">资源</p>
      <nav class="side-nav">
        <router-link to="/media" class="side-link" :class="{ active: isMediaActive }">
          <el-icon><Picture /></el-icon>
          <span>素材管理</span>
        </router-link>
      </nav>
      <p class="side-note">视频 · 图片 · 素材库</p>

      <p class="side-label">线索</p>
      <nav class="side-nav">
        <router-link
          to="/submissions"
          class="side-link"
          :class="{ active: isSubmissionsActive }"
        >
          <el-icon><Message /></el-icon>
          <span>表单留言</span>
          <span
            v-if="newSubmissionBadge"
            class="side-badge"
            :title="`${pagesStore.newSubmissionCount} 条待处理留言`"
          >
            {{ newSubmissionBadge }}
          </span>
        </router-link>
      </nav>
      <p class="side-note">联系表单提交 · 跟进状态</p>

      <p class="side-label">设置</p>
      <nav class="side-nav">
        <router-link
          to="/settings"
          class="side-link"
          :class="{ active: isSettingsActive }"
        >
          <el-icon><Setting /></el-icon>
          <span>基础设置</span>
        </router-link>
      </nav>
      <p class="side-note">后台登录 · 展示文案</p>
    </aside>

    <div class="admin-main">
      <header class="admin-topbar">
        <div class="crumb">
          <span>工作台</span>
          <span>/</span>
          <strong>{{ route.meta.title || '站点编排' }}</strong>
          <el-tag v-if="pagesStore.loadError" type="danger" size="small" effect="plain">
            数据加载失败
          </el-tag>
          <el-tag v-else-if="pagesStore.storageError" type="danger" size="small" effect="plain">
            设置未保存
          </el-tag>
        </div>
        <div class="top-user">
          <span class="avatar user-avatar">
            <img v-if="avatarUrl" :src="avatarUrl" alt="" />
            <template v-else>{{ avatarText }}</template>
          </span>
          <div class="top-user-text">
            <strong>{{ displayName }}</strong>
            <span>管理员</span>
          </div>
          <router-link to="/settings/account" class="top-user-action" title="账号与安全">
            <el-icon><User /></el-icon>
          </router-link>
          <button class="top-user-action" title="退出登录" @click="handleLogout">
            <el-icon><SwitchButton /></el-icon>
          </button>
        </div>
      </header>

      <main class="admin-content">
        <router-view />
      </main>
    </div>

    <el-alert
      v-if="pagesStore.loadError"
      class="storage-alert"
      title="站点数据加载失败，请刷新页面重试。"
      type="error"
      :closable="false"
      show-icon
    />
    <el-alert
      v-else-if="pagesStore.storageError"
      class="storage-alert"
      :title="pagesStore.storageError"
      type="warning"
      :closable="false"
      show-icon
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Document,
  Message,
  Picture,
  Setting,
  SwitchButton,
  User
} from '@element-plus/icons-vue'
import { usePagesStore } from '@/stores/pages'
import { logout, profileRef, sessionRef } from '@/services/auth'

const route = useRoute()
const router = useRouter()
const pagesStore = usePagesStore()
const displayName = computed(
  () => profileRef.value.name.trim() || sessionRef.value?.name?.trim() || '管理员'
)
const avatarText = computed(() => displayName.value.charAt(0) || '管')
const avatarUrl = computed(() => profileRef.value.avatar)

const isPagesActive = computed(
  () => route.path === '/pages' || route.path.startsWith('/pages/')
)
const isSubmissionsActive = computed(
  () => route.path === '/submissions' || route.path.startsWith('/submissions/')
)
const isMediaActive = computed(
  () => route.path === '/media' || route.path.startsWith('/media/')
)

const newSubmissionBadge = computed(() => {
  const count = pagesStore.newSubmissionCount
  if (count <= 0) return ''
  return count > 99 ? '99+' : String(count)
})

/** 红点轮询间隔：后台是运营工具，30 秒足够，不必做成实时大屏。 */
const BADGE_POLL_MS = 30_000
let badgeTimer: ReturnType<typeof setInterval> | null = null

function startBadgePolling() {
  if (badgeTimer) return
  badgeTimer = setInterval(() => {
    void pagesStore.refreshSubmissionBadge()
  }, BADGE_POLL_MS)
}

function stopBadgePolling() {
  if (!badgeTimer) return
  clearInterval(badgeTimer)
  badgeTimer = null
}

/** 标签页不可见时停止轮询；切回时立刻刷新一次，再恢复轮询。 */
function handleVisibilityChange() {
  if (document.visibilityState === 'visible') {
    void pagesStore.refreshSubmissionBadge()
    startBadgePolling()
    return
  }
  stopBadgePolling()
}

onMounted(() => {
  // 外壳只取自己显示的两个数字，不加载页面内容。
  void pagesStore.loadPageCount()
  void pagesStore.refreshSubmissionBadge()
  if (document.visibilityState === 'visible') startBadgePolling()
  document.addEventListener('visibilitychange', handleVisibilityChange)
})

onBeforeUnmount(() => {
  stopBadgePolling()
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})
const isSettingsActive = computed(
  () => route.path === '/settings' || route.path.startsWith('/settings/')
)

async function handleLogout() {
  try {
    await ElMessageBox.confirm('确定退出当前账号吗？', '退出登录', {
      type: 'warning',
      confirmButtonText: '退出登录',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }

  await logout()
  ElMessage.success('已退出登录')
  await router.replace({ name: 'login' })
}
</script>

<style scoped>
.storage-alert {
  position: fixed;
  right: 20px;
  bottom: 20px;
  width: 360px;
  z-index: 100;
}

.top-user {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-avatar {
  width: 30px;
  height: 30px;
  overflow: hidden;
}

.user-avatar img {
  display: block;
  width: 100%;
  height: 100%;
  border-radius: 50%;
  object-fit: cover;
}

.top-user-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  line-height: 1.35;
  margin-right: 4px;
}

.top-user-text strong {
  color: var(--adm-ink);
  font-size: 12px;
  font-weight: 600;
}

.top-user-text span {
  font-size: 10px;
  color: var(--adm-muted-2);
}

.top-user-action {
  width: 28px;
  height: 28px;
  flex: none;
  display: grid;
  place-items: center;
  padding: 0;
  border: 1px solid var(--adm-line);
  border-radius: 8px;
  background: transparent;
  color: var(--adm-muted);
  text-decoration: none;
  cursor: pointer;
  transition: background 0.18s ease, color 0.18s ease;
}

.top-user-action:hover {
  background: #fff;
  color: var(--adm-accent);
}

@media (max-width: 720px) {
  .top-user-text {
    display: none;
  }
}
</style>
