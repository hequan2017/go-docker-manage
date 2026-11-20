<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api'

const nodes = ref([])
const nodeId = ref(null)
const images = ref([])
const loading = ref(false)
const showPull = ref(false)
const pullForm = reactive({ ref: 'traefik:v3' })
const pulling = ref(false)

const nodeOptions = computed(() => nodes.value.map((n) => ({ value: n.id, label: n.name })))

async function loadNodes() {
  nodes.value = (await http.get('/nodes')) || []
  if (!nodeId.value && nodes.value.length) nodeId.value = nodes.value[0].id
}

async function load() {
  if (!nodeId.value) return
  loading.value = true
  try {
    images.value = (await http.get(`/nodes/${nodeId.value}/images`)) || []
  } finally {
    loading.value = false
  }
}

async function pull() {
  if (!pullForm.ref) return
  pulling.value = true
  try {
    const r = await http.post(`/nodes/${nodeId.value}/images/pull`, { ref: pullForm.ref })
    ElMessage.success(r.status || '拉取完成')
    showPull.value = false
    load()
  } finally {
    pulling.value = false
  }
}

async function remove(row) {
  await ElMessageBox.confirm(`确定删除镜像「${row.repoTags[0]}」?`, '确认', { type: 'warning' })
  await http.delete(`/nodes/${nodeId.value}/images`, { params: { ref: row.repoTags[0], force: true } })
  ElMessage.success('已删除')
  load()
}

onMounted(async () => {
  await loadNodes()
  load()
})
</script>

<template>
  <el-card>
    <template #header>
      <div class="bar">
        <div class="filters">
          <el-select v-model="nodeId" placeholder="选择节点" style="width: 180px" @change="load">
            <el-option v-for="o in nodeOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
          <el-button @click="load">刷新</el-button>
        </div>
        <el-button type="primary" @click="showPull = true">⬇ 拉取镜像</el-button>
      </div>
    </template>

    <el-empty v-if="!nodes.length" description="还没有节点,请先到「节点管理」添加" />
    <el-table v-else :data="images" v-loading="loading" stripe>
      <el-table-column label="镜像" min-width="240">
        <template #default="{ row }">
          <b>{{ row.repoTags[0] }}</b>
          <div class="iid">{{ row.id.replace('sha256:', '').slice(0, 20) }}…</div>
        </template>
      </el-table-column>
      <el-table-column prop="sizeText" label="大小" width="110" />
      <el-table-column prop="created" label="创建时间" width="170" />
      <el-table-column label="操作" width="110" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="danger" plain @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="showPull" title="拉取镜像" width="440">
    <el-form label-width="80" @keyup.enter="pull">
      <el-form-item label="镜像名">
        <el-input v-model="pullForm.ref" placeholder="如 nginx:alpine" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="showPull = false">取消</el-button>
      <el-button type="primary" :loading="pulling" @click="pull">拉取</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.bar { display: flex; justify-content: space-between; align-items: center; }
.filters { display: flex; gap: 12px; align-items: center; }
.iid { color: #909399; font-size: 12px; font-family: Consolas, monospace; }
</style>
