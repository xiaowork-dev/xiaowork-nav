<script setup lang="ts">
import { onMounted, ref } from 'vue'
import * as adminApi from '@/api/admin'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { errorMessage, useToast } from '@/composables/useToast'
import type { Backup, ConflictStrategy, ExportBundle, ImportPreview, ImportResult, UpdateInfo } from '@/types/api'

const toast = useToast()

const exporting = ref(false)
const previewing = ref(false)
const importing = ref(false)
const preview = ref<ImportPreview | null>(null)
const importResult = ref<ImportResult | null>(null)
const strategy = ref<ConflictStrategy>('skip')
const pendingBundle = ref<ExportBundle | null>(null)
const fileName = ref('')

const backups = ref<Backup[]>([])
const backupsLoading = ref(false)
const pendingRestore = ref<Backup | null>(null)
const restoring = ref(false)

const updateInfo = ref<UpdateInfo | null>(null)
const checking = ref(false)

async function exportData(): Promise<void> {
  exporting.value = true
  try {
    const bundle = await adminApi.exportData()
    const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = `xiaowork-nav-export-${new Date().toISOString().slice(0, 10)}.json`
    document.body.appendChild(anchor)
    anchor.click()
    document.body.removeChild(anchor)
    URL.revokeObjectURL(url)
    toast.success('已导出数据文件')
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    exporting.value = false
  }
}

async function onFileSelected(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  previewing.value = true
  preview.value = null
  importResult.value = null
  pendingBundle.value = null
  try {
    const text = await file.text()
    const parsed = JSON.parse(text) as ExportBundle
    pendingBundle.value = parsed
    fileName.value = file.name
    preview.value = await adminApi.previewImport(parsed)
  } catch (e) {
    toast.error(`无法解析导入文件：${errorMessage(e)}`)
  } finally {
    previewing.value = false
    input.value = ''
  }
}

async function runImport(): Promise<void> {
  if (!pendingBundle.value) return
  importing.value = true
  try {
    importResult.value = await adminApi.runImport(pendingBundle.value, strategy.value)
    toast.success('导入完成')
    preview.value = null
    pendingBundle.value = null
    await loadBackups()
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    importing.value = false
  }
}

async function loadBackups(): Promise<void> {
  backupsLoading.value = true
  try {
    backups.value = (await adminApi.listBackups()).items
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    backupsLoading.value = false
  }
}

async function createBackup(): Promise<void> {
  try {
    await adminApi.createBackup()
    toast.success('备份已创建')
    await loadBackups()
  } catch (e) {
    toast.error(errorMessage(e))
  }
}

async function confirmRestore(): Promise<void> {
  if (!pendingRestore.value) return
  restoring.value = true
  try {
    await adminApi.restoreBackup(pendingRestore.value.id)
    toast.success('已从备份恢复')
    pendingRestore.value = null
    await loadBackups()
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    restoring.value = false
  }
}

async function checkUpdate(): Promise<void> {
  checking.value = true
  try {
    updateInfo.value = await adminApi.checkUpdate()
    if (!updateInfo.value.update_available) {
      toast.info('当前已是最新版本')
    }
  } catch (e) {
    toast.error(errorMessage(e))
  } finally {
    checking.value = false
  }
}

function formatSize(bytes?: number): string {
  if (!bytes) return '—'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

onMounted(loadBackups)
</script>

<template>
  <div class="card">
    <div class="card__head">
      <h2 class="card__title">导出数据</h2>
    </div>
    <p class="hint" style="margin-top: 0">导出全部分类、站点与设置为一个 JSON 文件。</p>
    <button type="button" class="btn btn--primary" :disabled="exporting" @click="exportData">
      {{ exporting ? '导出中…' : '导出为 JSON' }}
    </button>
  </div>

  <div class="card">
    <div class="card__head">
      <h2 class="card__title">导入数据</h2>
    </div>
    <p class="hint" style="margin-top: 0">
      支持本系统导出的 JSON 与旧站数据（<code class="code">legacy-v1</code>）。导入默认不覆盖已有数据。
    </p>

    <input type="file" accept="application/json,.json" @change="onFileSelected" />
    <p v-if="previewing" class="dim">正在解析并预览…</p>

    <div v-if="preview" class="panel" style="margin-top: 14px">
      <p style="margin-top: 0">
        文件：<strong>{{ fileName }}</strong> · 格式：<code class="code">{{ preview.format }}</code>
      </p>
      <p class="muted" style="font-size: 0.86rem">
        分类：新增 {{ preview.categories_new }} / 冲突 {{ preview.categories_conflict }}；
        站点：新增 {{ preview.bookmarks_new }} / 冲突 {{ preview.bookmarks_conflict }}
      </p>
      <ul v-if="preview.warnings?.length" class="dim" style="font-size: 0.82rem">
        <li v-for="(warning, index) in preview.warnings" :key="index">{{ warning }}</li>
      </ul>

      <div class="field">
        <label for="imp-strategy">冲突处理策略</label>
        <select id="imp-strategy" v-model="strategy" class="select">
          <option value="skip">跳过已存在的条目（默认）</option>
          <option value="overwrite">覆盖已存在的条目</option>
          <option value="rename">保留两者（重命名导入项）</option>
        </select>
      </div>

      <button type="button" class="btn btn--primary" :disabled="importing" @click="runImport">
        {{ importing ? '导入中…' : '确认导入' }}
      </button>
    </div>

    <div v-if="importResult" class="panel" style="margin-top: 14px">
      <p style="margin: 0">
        导入完成：分类 {{ importResult.categories_imported }}，站点 {{ importResult.bookmarks_imported }}，跳过
        {{ importResult.skipped }}。
      </p>
    </div>
  </div>

  <div class="card">
    <div class="card__head">
      <h2 class="card__title">备份与恢复</h2>
      <button type="button" class="btn btn--sm btn--primary" @click="createBackup">立即备份</button>
    </div>

    <div v-if="backupsLoading" class="empty">加载中…</div>
    <div v-else-if="!backups.length" class="empty">暂无备份快照。</div>

    <div v-else class="list">
      <div v-for="backup in backups" :key="backup.id" class="list-item">
        <div class="list-item__main">
          <div class="list-item__title">{{ backup.id }}</div>
          <div class="list-item__meta">
            {{ backup.created_at }} · {{ formatSize(backup.size) }}
            <template v-if="backup.schema_version"> · 结构版本 {{ backup.schema_version }}</template>
          </div>
        </div>
        <div class="list-item__actions">
          <button type="button" class="btn btn--sm" @click="pendingRestore = backup">恢复</button>
        </div>
      </div>
    </div>
  </div>

  <div class="card">
    <div class="card__head">
      <h2 class="card__title">版本信息</h2>
      <button type="button" class="btn btn--sm" :disabled="checking" @click="checkUpdate">
        {{ checking ? '检查中…' : '检查更新' }}
      </button>
    </div>
    <p v-if="updateInfo" class="muted" style="font-size: 0.86rem">
      当前版本 <code class="code">{{ updateInfo.current_version }}</code> · 最新版本
      <code class="code">{{ updateInfo.latest_version }}</code>
      <template v-if="updateInfo.update_available">
        · <a :href="updateInfo.release_url" target="_blank" rel="noopener noreferrer">查看发行说明</a>
      </template>
    </p>
    <p v-else class="hint" style="margin: 0">
      更新检查为只读操作；如需在服务器上升级，请在部署目录运行更新脚本。
    </p>
  </div>

  <ConfirmDialog
    v-if="pendingRestore"
    title="恢复备份"
    :message="`确定用备份“${pendingRestore.id}”覆盖当前数据吗？当前数据会先自动备份一次。`"
    confirm-label="恢复"
    :busy="restoring"
    @cancel="pendingRestore = null"
    @confirm="confirmRestore"
  />
</template>
