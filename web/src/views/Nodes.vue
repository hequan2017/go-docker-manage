<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api'

const nodes = ref([])
const loading = ref(false)
const showCreate = ref(false)
const form = reactive({
  name: 'mock-node-1',
  host: '',
  dockerApiUrl: 'tcp://mock-dockerd:2375',
  description: '模拟 Docker 节点',
})

async function load() {
  loading.value = true
  try {
    nodes.value = (await http.get('/nodes')) || []
  } finally {
    loading.value = false
  }
}

async function create() {
  if (!form.name || !form.dockerApiUrl) {
    ElMessage.warning('节点名与 Docker API 地址必填')
    return
  }
  await http.post('/nodes', { ...form })
  ElMessage.success('节点已创建,建议立即做一次健康检查')
  showCreate.value = false
  load()
}

async function healthCheck(row) {
  const r = await http.post(`/nodes/${row.id}/health`)
  if (r.ok) {
    ElMessage.success(`健康: Docker ${r.dockerVersion || '?'} (${r.message})`)
  } else {
    ElMessage.error(`异常: ${r.message}`)
  }
  load()
}

async function remove(row) {
  await ElMessageBox.confirm(`确定删除节点「${row.name}」?`, '确认', { type: 'warning' })
  await http.delete(`/nodes/${row.id}`)
  ElMessage.success('已删除')
  load()
}

onMounted(load)
</script>

<template>
  <el-card>
    <template #header>
      <div class="bar">
        <span>Docker 节点(模拟对接:地址指向 mock-dockerd;接入真实机器时填 tcp://IP:2375)</span>
        <el-button type="primary" @click="showCreate = true">+ 添加节点</el-button>
      </div>
    </template>
    <el-table :data="nodes" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="60" />
      <el-table-column prop="name" label="名称" min-width="120" />
      <el-table-column prop="host" label="主机" min-width="110" />
      <el-table-column prop="dockerApiUrl" label="Docker API" min-width="200" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'healthy' ? 'success' : row.status === 'unhealthy' ? 'danger' : 'info'" size="small">
            {{ row.status }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" min-width="140" />
      <el-table-column prop="createdAt" label="创建时间" width="170" />
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <el-button size="small" type="primary" plain @click="healthCheck(row)">健康检查</el-button>
          <el-button size="small" type="danger" plain @click="remove(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-card>

  <el-dialog v-model="showCreate" title="添加节点" width="480">
    <el-form label-width="110">
      <el-form-item label="节点名称" required>
        <el-input v-model="form.name" placeholder="如 worker-1" />
      </el-form-item>
      <el-form-item label="主机地址">
        <el-input v-model="form.host" placeholder="如 192.168.112.138" />
      </el-form-item>
      <el-form-item label="Docker API" required>
        <el-input v-model="form.dockerApiUrl" placeholder="tcp://host:2375 或 unix:///var/run/docker.sock" />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="showCreate = false">取消</el-button>
      <el-button type="primary" @click="create">创建</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.bar { display: flex; justify-content: space-between; align-items: center; }
</style>
