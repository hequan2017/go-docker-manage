<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api'

const nodes = ref([])
const nodeId = ref(null)
const all = ref(true)
const containers = ref([])
const loading = ref(false)

const showCreate = ref(false)
const createForm = reactive({ name: '', image: '', portsText: '', envText: '' })
const logsVisible = ref(false)
const statsVisible = ref(false)
const logsText = ref('')
const statsData = ref(null)
const activeContainer = ref(null)
let timer = null

const nodeOptions = computed(() => nodes.value.map((n) => ({ value: n.id, label: n.name })))

async function loadNodes() {
  nodes.value = (await http.get('/nodes')) || []
  if (!nodeId.value && nodes.value.length) nodeId.value = nodes.value[0].id
}

async function load() {
  if (!nodeId.value) return
  loading.value = true
  try {
    containers.value = (await http.get(`/nodes/${nodeId.value}/containers`, { params: { all: all.value } })) || []
  } finally {
    loading.value = false
  }
}

async function action(row, act) {
  const msg = await http.post(`/nodes/${nodeId.value}/containers/${row.id}/${act}`)
  ElMessage.success(msg || `${act} 完成`)
  load()
}

async function remove(row) {
  const force = row.state === 'running'
  await ElMessageBox.confirm(
    force ? `容器「${row.name}」运行中,将强制删除,确认?` : `确定删除容器「${row.name}」?`,
    '确认',
    { type: 'warning' },
  )
  await http.delete(`/nodes/${nodeId.value}/containers/${row.id}`, { params: { force } })
  ElMessage.success('已删除')
  load()
}

async function create() {
  if (!createForm.image) {
    ElMessage.warning('镜像名必填,如 nginx:alpine')
    return
  }
  const env = createForm.envText.split('\n').map((s) => s.trim()).filter(Boolean)
  const ports = createForm.portsText
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean)
    .map((line) => {
      // 格式: 宿主端口:容器端口[:协议],如 8080:80 或 53:53:udp
      const [hostPort, containerPort, protocol = 'tcp'] = line.split(':')
      return { hostPort, containerPort, protocol }
    })
  const r = await http.post(`/nodes/${nodeId.value}/containers`, {
    name: createForm.name,
    image: createForm.image,
    env,
    ports,
  })
  ElMessage.success(`容器已创建:${r.containerId.slice(0, 12)},点击启动使其运行`)
  showCreate.value = false
  Object.assign(createForm, { name: '', image: '', portsText: '', envText: '' })
  load()
}

async function showLogs(row) {
  activeContainer.value = row
  logsText.value = '加载中…'
  logsVisible.value = true
  const logs = await http.get(`/nodes/${nodeId.value}/containers/${row.id}/logs`, {
    params: { tail: 200, timestamps: true },
  })
  logsText.value = (logs || '(空)').split('\n').join('\n')
}

async function showStats(row) {
  activeContainer.value = row
  statsVisible.value = true
  statsData.value = await http.get(`/nodes/${nodeId.value}/containers/${row.id}/stats`)
}

function stateColor(state) {
  return { running: 'success', exited: 'danger', created: 'info', paused: 'warning' }[state] || 'info'
}

onMounted(async () => {
  await loadNodes()
  load()
  timer = setInterval(() => load(), 8000)
})
onUnmounted(() => clearInterval(timer))
</script>

<template>
  <el-card>
    <template #header>
      <div class="bar">
        <div class="filters">
          <el-select v-model="nodeId" placeholder="选择节点" style="width: 180px" @change="load">
            <el-option v-for="o in nodeOptions" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
          <el-checkbox v-model="all" @change="load">显示已停止</el-checkbox>
          <el-button @click="load">刷新</el-button>
        </div>
        <el-button type="primary" @click="showCreate = true">+ 创建容器</el-button>
      </div>
    </template>

    <el-empty v-if="!nodes.length" description="还没有节点,请先到「节点管理」添加" />
    <el-table v-else :data="containers" v-loading="loading" stripe>
      <el-table-column label="名称" min-width="130">
        <template #default="{ row }">
          <b>{{ row.name }}</b>
          <div class="cid">{{ row.id.slice(0, 12) }}</div>
        </template>
      </el-table-column>
      <el-table-column prop="image" label="镜像" min-width="180" />
      <el-table-column label="状态" width="110">
        <template #default="{ row }">
          <el-tag :type="stateColor(row.state)" size="small">{{ row.state }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="Status" min-width="150" />
      <el-table-column label="端口" min-width="150">
        <template #default="{ row }">
          <span v-for="p in row.ports" :key="p.host + p.container" class="port">
            {{ p.host || '*'}} → {{ p.container }}/{{ p.protocol }}
          </span>
        </template>
      </el-table-column>
      <el-table-column prop="createdAt" label="创建时间" width="165" />
      <el-table-column label="操作" width="320" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="success" plain :disabled="row.state === 'running'" @click="action(row, 'start')">启动</el-button>
          <el-button size="small" type="warning" plain :disabled="row.state !== 'running'" @click="action(row, 'stop')">停止</el-button>
          <el-button size="small" plain @click="action(row, 'restart')">重启</el-button>
          <el-button size="small" @click="showStats(row)">Stats</el-button>
          <el-button size="small" @click="showLogs(row)">日志</el-button>
          <el-button size="small" type="danger" plain @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="showCreate" title="创建容器" width="520">
    <el-form label-width="90">
      <el-form-item label="名称">
        <el-input v-model="createForm.name" placeholder="可选,如 my-nginx" />
      </el-form-item>
      <el-form-item label="镜像" required>
        <el-input v-model="createForm.image" placeholder="nginx:alpine(需先在镜像页拉取)" />
      </el-form-item>
      <el-form-item label="端口映射">
        <el-input v-model="createForm.portsText" type="textarea" :rows="3" placeholder="每行一条:宿主端口:容器端口[:协议]&#10;如 8080:80" />
      </el-form-item>
      <el-form-item label="环境变量">
        <el-input v-model="createForm.envText" type="textarea" :rows="2" placeholder="每行一条 KEY=value" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="showCreate = false">取消</el-button>
      <el-button type="primary" @click="create">创建</el-button>
    </template>
  </el-dialog>

  <el-drawer v-model="logsVisible" :title="`日志 - ${activeContainer?.name || ''}`" size="50%">
    <pre class="logs">{{ logsText }}</pre>
  </el-drawer>

  <el-dialog v-model="statsVisible" :title="`资源占用 - ${activeContainer?.name || ''}`" width="440">
    <el-descriptions :column="2" border v-if="statsData">
      <el-descriptions-item label="CPU">{{ statsData.cpuPercent.toFixed(2) }} %</el-descriptions-item>
      <el-descriptions-item label="进程数">{{ statsData.pids }}</el-descriptions-item>
      <el-descriptions-item label="内存">
        {{ statsData.memUsageMb.toFixed(1) }} / {{ statsData.memLimitMb.toFixed(0) }} MB
      </el-descriptions-item>
      <el-descriptions-item label="网络 ↓{{ statsData.netRxMb.toFixed(1) }}MB ↑{{ statsData.netTxMb.toFixed(1) }}MB">—</el-descriptions-item>
    </el-descriptions>
  </el-dialog>
</template>

<style scoped>
.bar { display: flex; justify-content: space-between; align-items: center; }
.filters { display: flex; gap: 12px; align-items: center; }
.cid { color: #909399; font-size: 12px; font-family: Consolas, monospace; }
.port { display: inline-block; margin-right: 6px; font-size: 12px; font-family: Consolas, monospace; }
.logs {
  background: #1d2939; color: #d4e7ff; padding: 14px; border-radius: 6px;
  font-size: 12px; line-height: 1.7; white-space: pre-wrap; word-break: break-all;
  max-height: calc(100vh - 140px); overflow: auto;
}
</style>
