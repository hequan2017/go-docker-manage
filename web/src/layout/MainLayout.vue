<script setup>
import { useRouter } from 'vue-router'
import { useAuthStore } from '../store/auth'

const router = useRouter()
const auth = useAuthStore()

function onLogout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <el-container style="height: 100%">
    <el-aside width="210px" style="background: #1d2939">
      <div class="logo">🐳 Docker 管理</div>
      <el-menu
        :default-active="$route.path"
        router
        background-color="#1d2939"
        text-color="#a7b1c2"
        active-text-color="#409eff"
        style="border-right: none"
      >
        <el-menu-item index="/dashboard">📊 仪表盘</el-menu-item>
        <el-menu-item index="/nodes">🖥️ 节点管理</el-menu-item>
        <el-menu-item index="/containers">📦 容器管理</el-menu-item>
        <el-menu-item index="/images">🧩 镜像管理</el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <div class="title">{{ $route.meta.title }}</div>
        <div class="right">
          <span class="user">👤 {{ auth.nickname }}</span>
          <el-button size="small" @click="onLogout">退出</el-button>
        </div>
      </el-header>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<style scoped>
.logo {
  color: #fff;
  font-size: 17px;
  font-weight: 600;
  text-align: center;
  line-height: 60px;
  border-bottom: 1px solid #2b3a4b;
}
.header {
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-shadow: 0 1px 4px rgba(0, 21, 41, 0.08);
}
.title { font-size: 16px; font-weight: 600; }
.right { display: flex; align-items: center; gap: 12px; }
.user { color: #606266; font-size: 14px; }
</style>
