<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import http from '../api'

const stats = ref(null)
const loading = ref(false)
let timer = null

async function load() {
  loading.value = true
  try {
    stats.value = await http.get('/dashboard')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  load()
  timer = setInterval(load, 10000)
})
onUnmounted(() => clearInterval(timer))
</script>

<template>
  <div v-loading="loading">
    <el-row :gutter="16">
      <el-col :span="4" v-for="card in [
        { label: '节点总数', value: stats?.nodeTotal, icon: '🖥️', color: '#409eff' },
        { label: '健康节点', value: stats?.nodeHealthy, icon: '✅', color: '#67c23a' },
        { label: '容器总数', value: stats?.containerTotal, icon: '📦', color: '#909399' },
        { label: '运行中容器', value: stats?.containerRunning, icon: '🟢', color: '#67c23a' },
        { label: '镜像总数', value: stats?.imageTotal, icon: '🧩', color: '#e6a23c' },
      ]" :key="card.label">
        <el-card class="stat-card">
          <div class="stat">
            <span class="icon">{{ card.icon }}</span>
            <div>
              <div class="value" :style="{ color: card.color }">{{ card.value ?? '-' }}</div>
              <div class="label">{{ card.label }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card style="margin-top: 16px">
      <template #header>节点概览</template>
      <el-table :data="stats?.nodes || []" stripe>
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" label="节点" min-width="140" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'healthy' ? 'success' : row.status === 'unhealthy' ? 'danger' : 'info'" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="containers" label="容器数" width="90" />
        <el-table-column prop="running" label="运行中" width="90" />
        <el-table-column prop="images" label="镜像数" width="90" />
        <el-table-column prop="error" label="异常" min-width="140" />
      </el-table>
    </el-card>
  </div>
</template>

<style scoped>
.stat-card :deep(.el-card__body) { padding: 14px; }
.stat { display: flex; align-items: center; gap: 12px; }
.icon { font-size: 26px; }
.value { font-size: 26px; font-weight: 700; line-height: 1.1; }
.label { color: #909399; font-size: 13px; }
</style>
