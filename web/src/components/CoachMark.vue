<template>
  <Teleport to="body">
    <div v-if="open" class="coach-mask" @click.self="close">
      <div class="coach-spot" :style="spotStyle"></div>
      <div v-if="step" class="coach-bub" :style="bubStyle">
        <span class="cs-tag"><el-icon :size="13"><Spotlight /></el-icon>界面导览 {{ idx + 1 }} / {{ steps.length }}</span>
        <h4>{{ step.t }}</h4>
        <p>{{ step.d }}</p>
        <div class="cb-nav">
          <el-button size="small" text @click="close">跳过</el-button>
          <span class="grow" />
          <el-button v-if="idx > 0" size="small" @click="idx--">上一步</el-button>
          <el-button size="small" type="primary" @click="next">{{ idx === steps.length - 1 ? '完成' : '下一步' }}</el-button>
        </div>
        <div class="cb-dots"><i v-for="(s, i) in steps" :key="i" :class="{ on: i <= idx }"></i></div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Spotlight } from 'lucide-vue-next'

interface Step { sel: string; side: 'right' | 'bottom'; t: string; d: string }

const steps: Step[] = [
  { sel: '.tree-panel', side: 'right', t: '这就是目录本身',
    d: '左侧的树是 LDAP 目录（DIT）：部门、人员、组都是树上的条目。你在 phpLDAPadmin、LAM 里看到的是同一棵树——因为目录即真相。' },
  { sel: '#tour-new-ou', side: 'right', t: '新建部门',
    d: '部门（OU）是树的枝干。这里新建的部门立刻出现在其他 LDAP 客户端里，反之亦然。' },
  { sel: 'a[href="/people"]', side: 'right', t: '日常入口：组织与人员',
    d: '左树右表。支持行内加人、勾选批量调动、自定义列；Excel 批量导入在「导入导出」页。' },
  { sel: '.top-search', side: 'bottom', t: '全局搜索',
    d: '输入姓名或账号，下拉建议直接打开人员详情——不用先找到部门。' },
  { sel: 'a[href="/help"]', side: 'right', t: '帮助中心',
    d: '对接堡垒机 / 零信任该填什么、LDAP 概念看不懂、术语对照——都在这里。导览完成！' },
]

const open = defineModel<boolean>({ default: false })
const idx = ref(0)
const rect = ref<DOMRect | null>(null)

const step = computed(() => steps[idx.value] ?? null)
const spotStyle = computed(() => {
  const r = rect.value
  if (!r) return {}
  const pad = 6
  return {
    left: `${r.left - pad}px`, top: `${r.top - pad}px`,
    width: `${r.width + pad * 2}px`, height: `${r.height + pad * 2}px`,
  }
})
const bubStyle = computed(() => {
  const r = rect.value
  if (!r) return { top: '40vh', left: '40vw' }
  const bw = 310, bh = 190
  let left: number, top: number
  if (r.right + bw + 24 < window.innerWidth) {
    left = r.right + 24
    top = Math.min(Math.max(r.top, 12), window.innerHeight - bh - 12)
  } else {
    left = Math.min(Math.max(r.left, 12), window.innerWidth - bw - 12)
    top = r.bottom + bh + 16 < window.innerHeight ? r.bottom + 16 : Math.max(12, r.top - bh - 16)
  }
  return { left: `${left}px`, top: `${top}px` }
})

function measure() {
  const el = step.value ? document.querySelector(step.value.sel) : null
  if (!el) { rect.value = null; return }
  const r = el.getBoundingClientRect()
  rect.value = r.width || r.height ? r : null
}
function next() {
  if (idx.value === steps.length - 1) {
    close()
    ElMessage.success({ message: '导览完成。之后随时可以在帮助中心重看', duration: 2500 })
  } else idx.value++
}
function close() { open.value = false }

watch([open, idx], ([o]) => {
  if (o) { idx.value = Math.min(idx.value, steps.length - 1); requestAnimationFrame(measure) }
})
onMounted(() => {
  window.addEventListener('resize', measure)
  window.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', measure)
  window.removeEventListener('keydown', onKey)
})
function onKey(e: KeyboardEvent) { if (e.key === 'Escape' && open.value) close() }
</script>

<style scoped>
.coach-mask { position: fixed; inset: 0; z-index: 3000; }
.coach-spot { position: absolute; border-radius: 8px; box-shadow: 0 0 0 9999px rgba(15, 20, 40, 0.5), 0 0 0 2px var(--el-color-primary); pointer-events: none; }
.coach-bub { position: absolute; width: 310px; background: #fff; border-radius: 12px; box-shadow: 0 12px 32px rgba(31, 35, 41, 0.18); padding: 15px 17px; }
.cs-tag { display: inline-flex; align-items: center; gap: 6px; font-size: 11px; color: var(--el-color-primary); background: #f0f5ff; border-radius: 10px; padding: 3px 9px; margin-bottom: 8px; }
.coach-bub h4 { margin: 0 0 6px; font-size: 14px; }
.coach-bub p { margin: 0; font-size: 12.5px; color: #646a73; line-height: 1.75; }
.cb-nav { display: flex; align-items: center; gap: 8px; margin-top: 13px; }
.cb-nav .grow { flex: 1; }
.cb-dots { display: flex; gap: 4px; margin-top: 10px; }
.cb-dots i { width: 6px; height: 6px; border-radius: 50%; background: #e5e6eb; }
.cb-dots i.on { background: var(--el-color-primary); }
</style>
