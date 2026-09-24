<template>
  <div style="max-width: 1280px; margin: 0 auto">
    <div class="nl-page-head">
      <div>
        <h2>审计日志</h2>
        <div class="crumb">本机滚动留存 · 记录全部写操作与登录事件 · 密码等敏感值绝不记录</div>
      </div>
      <div class="ops">
        <el-button @click="refresh"><el-icon><RefreshCw /></el-icon>&nbsp;刷新</el-button>
        <el-button @click="exportCSV"><el-icon><Download /></el-icon>&nbsp;导出 CSV</el-button>
      </div>
    </div>
    <div class="nl-card nl-table-wrap">
      <div class="nl-toolbar">
        <el-input v-model="q" placeholder="搜索 DN / 操作人 / IP / 操作" clearable style="width: 260px">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <span class="spacer" />
        <span class="muted">最近 {{ limit }} 条</span>
      </div>
      <el-table :data="filtered" v-loading="loading" stripe>
        <el-table-column label="时间" width="160">
          <template #default="{ row }"><span class="mono" style="font-size: 11.5px">{{ fmt(row.ts) }}</span></template>
        </el-table-column>
        <el-table-column label="操作人" min-width="180">
          <template #default="{ row }"><span class="mono" style="font-size: 11.5px">{{ row.actor }}</span></template>
        </el-table-column>
        <el-table-column label="来源 IP" width="130">
          <template #default="{ row }"><span class="mono" style="font-size: 11.5px">{{ row.ip || '—' }}</span></template>
        </el-table-column>
        <el-table-column label="操作" width="130">
          <template #default="{ row }"><span class="nl-pill plain">{{ row.op }}</span></template>
        </el-table-column>
        <el-table-column label="目标 DN / 详情" min-width="240">
          <template #default="{ row }">
            <div class="mono" style="font-size: 11.5px">{{ row.dn || '—' }}</div>
            <div v-if="row.detail" class="muted" style="font-size: 11px">{{ row.detail }}</div>
          </template>
        </el-table-column>
        <el-table-column label="结果" width="80">
          <template #default="{ row }">
            <span class="nl-pill" :class="row.result === 'ok' ? 'ok' : 'dgr'"><span class="dot" />{{ row.result === 'ok' ? '成功' : '失败' }}</span>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Download, RefreshCw, Search } from 'lucide-vue-next'
import { errText, listAudit, type AuditRow } from '../api'

const rows = ref<AuditRow[]>([])
const loading = ref(false)
const q = ref('')
const limit = 100

const filtered = computed(() =>
  rows.value.filter((r) => !q.value || [r.actor, r.dn, r.ip, r.op, r.detail].join(' ').toLowerCase().includes(q.value.toLowerCase())))

async function refresh() {
  loading.value = true
  try {
    const res = await listAudit(limit)
    rows.value = res.events ?? []
  } catch (e) { ElMessage.error(errText(e)) } finally { loading.value = false }
}

function exportCSV() { window.open('/api/v1/audit/export?limit=1000', '_blank') }

function fmt(ts: string) {
  // 2026-09-21T06:30:00… → 本地易读
  if (!ts) return '—'
  return ts.replace('T', ' ').replace(/Z.*/, '')
}

onMounted(refresh)
</script>
