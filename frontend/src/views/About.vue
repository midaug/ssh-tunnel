<script lang="ts" setup>
import {ref, onMounted} from 'vue'
import * as App from '../../wailsjs/go/main/App'
import {main} from '../../wailsjs/go/models'

const info = ref<main.AppInfo>(new main.AppInfo())

onMounted(async () => {
  info.value = await App.GetAppInfo()
})

function openGitHub() {
  // 复用浏览器打开外部链接
  window.open(info.value.githubUrl, '_blank')
}
</script>

<template>
  <div class="page-title">
    <h2>关于</h2>
  </div>

  <div class="card about">
    <div class="app-head">
      <img src="../assets/images/logo-universal.png" alt="logo" class="logo">
      <div class="col">
        <strong class="app-name">SSH Tunnel</strong>
        <span class="version">版本 {{ info.version }}</span>
      </div>
    </div>
  </div>

  <div class="card">
    <div class="field">
      <label>作者</label>
      <div class="row">
        <span>{{ info.author }}</span>
        <span class="hint">{{ info.email }}</span>
      </div>
    </div>
    <div class="field">
      <label>GitHub 仓库</label>
      <div class="row">
        <span class="link" @click="openGitHub">{{ info.githubUrl }}</span>
      </div>
    </div>
    <div class="field">
      <label>开源协议</label>
      <div class="row">
        <span>{{ info.license }}</span>
      </div>
    </div>
  </div>

  <div class="card">
    <div class="col" style="gap:4px;">
      <span class="hint">SSH Tunnel 是一个基于 Wails + Vue 的 SSH 端口转发可视化管理工具。</span>
      <span class="hint">本软件基于 MIT 协议开源，可自由使用、修改和分发。</span>
    </div>
  </div>
</template>

<style scoped>
.about {
  display: flex;
}
.app-head {
  display: flex;
  align-items: center;
  gap: 16px;
}
.logo {
  width: 56px;
  height: 56px;
  border-radius: 12px;
}
.app-name {
  font-size: 20px;
}
.version {
  font-size: 13px;
  color: var(--hint-color, #888);
}
.link {
  color: #4f9cf9;
  cursor: pointer;
}
.link:hover {
  text-decoration: underline;
}
</style>
