<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import http from '../api'
import { useAuthStore } from '../store/auth'

const router = useRouter()
const auth = useAuthStore()
const loading = ref(false)
const isRegister = ref(false)

const form = reactive({ username: 'admin', password: '', nickname: '' })

async function submit() {
  if (!form.username || !form.password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    if (isRegister.value) {
      await http.post('/auth/register', {
        username: form.username,
        password: form.password,
        nickname: form.nickname,
      })
      ElMessage.success('注册成功,请登录')
      isRegister.value = false
      return
    }
    const data = await http.post('/auth/login', {
      username: form.username,
      password: form.password,
    })
    auth.setLogin(data)
    ElMessage.success('登录成功')
    router.push('/dashboard')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="page">
    <el-card class="card">
      <h2 class="title">🐳 Docker 管理平台</h2>
      <p class="sub">Go 微服务 · go-zero · Nacos</p>
      <el-form @keyup.enter="submit">
        <el-form-item>
          <el-input v-model="form.username" placeholder="用户名" size="large">
            <template #prepend>👤</template>
          </el-input>
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" placeholder="密码" size="large" show-password>
            <template #prepend>🔒</template>
          </el-input>
        </el-form-item>
        <el-form-item v-if="isRegister">
          <el-input v-model="form.nickname" placeholder="昵称(可选)" size="large">
            <template #prepend>💬</template>
          </el-input>
        </el-form-item>
        <el-button type="primary" size="large" style="width: 100%" :loading="loading" @click="submit">
          {{ isRegister ? '注 册' : '登 录' }}
        </el-button>
        <div class="switch">
          <el-link type="primary" @click="isRegister = !isRegister">
            {{ isRegister ? '已有账号?去登录' : '没有账号?注册一个' }}
          </el-link>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<style scoped>
.page {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1d2939 0%, #2b5876 100%);
}
.card { width: 380px; padding: 8px 12px; }
.title { text-align: center; margin: 8px 0 4px; }
.sub { text-align: center; color: #909399; margin: 0 0 18px; font-size: 13px; }
.switch { text-align: center; margin-top: 12px; }
</style>
