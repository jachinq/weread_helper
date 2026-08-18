<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { fetchSettings, saveSettings } from '../api'
import { applySiteTitle } from '../site'

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const ok = ref('')

const siteName = ref('')
const gatewayUrl = ref('')
const skillVersion = ref('')
const syncInterval = ref('')
const apiKey = ref('')
const apiKeyMasked = ref('')

async function load() {
  loading.value = true
  error.value = ''
  ok.value = ''
  try {
    const data = await fetchSettings()
    siteName.value = data.siteTitle || ''
    gatewayUrl.value = data.gatewayUrl || ''
    skillVersion.value = data.skillVersion || ''
    syncInterval.value = data.syncInterval || ''
    apiKeyMasked.value = data.apiKeyMasked || ''
    apiKey.value = ''
    applySiteTitle(data.siteTitle)
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

async function onSave() {
  saving.value = true
  error.value = ''
  ok.value = ''
  try {
    const data = await saveSettings({
      apiKey: apiKey.value.trim(),
      skillVersion: skillVersion.value.trim(),
      gatewayUrl: gatewayUrl.value.trim(),
      syncInterval: syncInterval.value.trim(),
      siteTitle: siteName.value.trim(),
    })
    apiKey.value = ''
    apiKeyMasked.value = data.apiKeyMasked || ''
    applySiteTitle(data.siteTitle)
    ok.value = '已保存，同步任务会使用新配置'
  } catch (e) {
    error.value = e instanceof Error ? e.message : '保存失败'
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <section :aria-busy="loading">
    <h2 class="page-title">设置</h2>
    <p class="muted">配置写入本地库。API Key 加密存储；输入框留空则不改现有密钥。</p>

    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <p v-if="ok" class="banner" role="status">{{ ok }}</p>

    <form class="settings-form" @submit.prevent="onSave">
      <fieldset>
        <legend>网站</legend>
        <label for="set-title">站点名称</label>
        <input id="set-title" v-model="siteName" type="text" autocomplete="off" maxlength="40" />
      </fieldset>

      <fieldset>
        <legend>微信读书 API</legend>
        <label for="set-key">API Key</label>
        <input
          id="set-key"
          v-model="apiKey"
          type="password"
          autocomplete="off"
          :placeholder="apiKeyMasked || 'wrk-…'"
        />
        <p class="field-hint">
          当前：{{ apiKeyMasked || '尚未配置' }} · 留空保存则保持原密钥
        </p>

        <label for="set-gw">Gateway URL</label>
        <input id="set-gw" v-model="gatewayUrl" type="url" autocomplete="off" />

        <label for="set-skill">skill_version</label>
        <input id="set-skill" v-model="skillVersion" type="text" autocomplete="off" />

        <label for="set-interval">同步间隔</label>
        <input id="set-interval" v-model="syncInterval" type="text" placeholder="6h" autocomplete="off" />
        <p class="field-hint">Go duration，例如 6h、30m</p>
      </fieldset>

      <button class="btn btn-solid" type="submit" :disabled="loading || saving">
        {{ saving ? '保存中…' : '保存' }}
      </button>
    </form>
  </section>
</template>
