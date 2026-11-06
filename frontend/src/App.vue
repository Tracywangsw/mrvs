<script setup>
import { onMounted, ref } from 'vue'
import { Calendar, Plus, TrendCharts } from '@element-plus/icons-vue'
import { checkHealth } from './api'

const health = ref(null)

onMounted(async () => {
  health.value = await checkHealth()
})
</script>

<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-mark">
          <svg width="22" height="22" viewBox="0 0 24 24" aria-hidden="true">
            <path
              d="M9 3h6v6h6v6h-6v6H9v-6H3V9h6V3z"
              fill="#0f6f66"
              opacity="0.16"
            />
            <path
              d="M10 2h4v6h6v4h-6v6h-4v-6H4v-4h6V2z"
              fill="#0b302c"
            />
          </svg>
        </div>
        <div>
          <div class="brand-title">MRVS</div>
          <div class="brand-sub">学术拜访管理</div>
        </div>
      </div>

      <nav class="nav">
        <div class="nav-label">工作台</div>
        <router-link to="/visits" class="nav-item">
          <el-icon><Calendar /></el-icon>
          <span>拜访列表</span>
        </router-link>
        <router-link to="/visits/new" class="nav-item">
          <el-icon><Plus /></el-icon>
          <span>新建拜访</span>
        </router-link>
        <router-link to="/dashboard" class="nav-item">
          <el-icon><TrendCharts /></el-icon>
          <span>产品月度看板</span>
        </router-link>
      </nav>

      <div class="sidebar-foot">
        <div
          class="health"
          :class="health === null ? '' : health ? 'online' : 'offline'"
        >
          <span class="dot"></span>
          <span>
            {{
              health === null
                ? '正在检测后端服务…'
                : health
                  ? '后端服务在线'
                  : '后端服务离线'
            }}
          </span>
        </div>
        <p class="foot-note">
          合规阈值 300 秒 / 500 米<br />
          时间统一按 Asia/Shanghai 展示
        </p>
      </div>
    </aside>

    <main class="main">
      <router-view />
    </main>
  </div>
</template>
