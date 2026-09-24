<template>
  <div class="learn-wrap">
    <aside class="nl-card lesson-nav">
      <div
        v-for="l in LESSONS" :key="l.key" class="lesson-item" :class="{ on: cur === l.key }"
        @click="cur = l.key"
      >
        <el-icon :size="16"><component :is="ICONS[l.icon] ?? CircleHelp" /></el-icon>
        <span>{{ l.title }}</span>
        <span v-if="l.lab" class="nl-pill info" style="transform: scale(0.86)">实验器</span>
      </div>
    </aside>
    <div class="nl-card lesson-body">
      <template v-for="l in LESSONS" :key="l.key">
        <template v-if="cur === l.key">
          <h3><el-icon :size="20"><component :is="ICONS[l.icon] ?? CircleHelp" /></el-icon>{{ l.title }}</h3>
          <div class="lsub">{{ l.sub }}</div>
          <!-- 课件 HTML 为前端内置常量（真实数据注入），非用户输入 -->
          <!-- eslint-disable-next-line vue/no-v-html -->
          <div class="prose" v-html="l.html(ctx)"></div>
          <FilterLab v-if="l.lab" />
          <div class="lfoot">
            <el-button v-if="prevOf(l.key)" text @click="cur = prevOf(l.key)!.key">
              ← {{ prevOf(l.key)!.title }}
            </el-button>
            <span class="grow" />
            <el-button v-if="nextOf(l.key)" type="primary" plain @click="cur = nextOf(l.key)!.key">
              下一课：{{ nextOf(l.key)!.title }} →
            </el-button>
            <el-button v-else type="primary" plain @click="$router.push({ path: '/help', query: { tab: 'connect' } })">
              去对接指引 →
            </el-button>
          </div>
        </template>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  CircleHelp, ListTree, Shapes, TableProperties, KeyRound, Filter,
  UsersRound, FileCode, Network, TriangleAlert,
} from 'lucide-vue-next'
import { me as apiMe, peopleList } from '../../api'
import { LESSONS, type LessonCtx } from '../../utils/lessons'
import FilterLab from './FilterLab.vue'

// lessons.ts 里的 icon 是字符串名 → 映射到已导入的 lucide 组件
//（<component :is> 传字符串解析不到局部导入的组件，必须给组件引用）
const ICONS: Record<string, unknown> = {
  ListTree, Shapes, TableProperties, KeyRound, Filter,
  UsersRound, FileCode, Network, TriangleAlert,
}

const cur = ref('dn')
const ctx = ref<LessonCtx>({ baseDN: 'dc=…', meDN: '', person: null })

const idx = computed(() => LESSONS.findIndex((l) => l.key === cur.value))
const prevOf = (k: string) => {
  const i = LESSONS.findIndex((l) => l.key === k)
  return i > 0 ? LESSONS[i - 1] : null
}
const nextOf = (k: string) => {
  const i = LESSONS.findIndex((l) => l.key === k)
  return i >= 0 && i < LESSONS.length - 1 ? LESSONS[i + 1] : null
}

onMounted(async () => {
  try {
    const m = await apiMe()
    ctx.value.baseDN = m.baseDN
    ctx.value.meDN = m.bindDN
  } catch { /* 忽略：课件用占位值 */ }
  try {
    const pl = await peopleList()
    const p = pl.people?.[0]
    if (p) {
      ctx.value.person = {
        dn: p.dn,
        cn: p.attrs?.cn?.[0] ?? '',
        uid: p.attrs?.uid?.[0] ?? '',
        ou: p.dn.split(',').slice(1).join(','),
        mail: p.attrs?.mail?.[0],
        title: p.attrs?.title?.[0],
      }
    }
  } catch { /* 忽略 */ }
})
</script>

<style scoped>
.learn-wrap { display: grid; grid-template-columns: 250px minmax(0, 1fr); gap: 14px; align-items: start; }
.lesson-nav { padding: 8px; display: flex; flex-direction: column; gap: 2px; }
.lesson-item { display: flex; align-items: center; gap: 9px; padding: 9px 10px; border-radius: 8px; font-size: 13px; color: #646a73; cursor: pointer; transition: 0.12s; }
.lesson-item:hover { background: #f2f3f5; color: #1f2329; }
.lesson-item.on { background: #f0f5ff; color: var(--el-color-primary); font-weight: 600; }
.lesson-item .nl-pill { margin-left: auto; }
.lesson-body { padding: 24px 28px; max-width: 920px; }
.lesson-body h3 { font-size: 17px; margin: 0 0 4px; display: flex; align-items: center; gap: 9px; }
.lesson-body h3 .el-icon { color: var(--el-color-primary); }
.lsub { font-size: 12px; color: #8f959e; margin-bottom: 16px; }
.lfoot { display: flex; align-items: center; margin-top: 20px; padding-top: 14px; border-top: 1px dashed var(--el-border-color-lighter); }
.lfoot .grow { flex: 1; }

/* 课件排版（v-html 内容）：lesson-body 深度选择器 */
.lesson-body :deep(.prose) { font-size: 13.5px; line-height: 1.9; color: #1f2329; }
.lesson-body :deep(.prose p) { margin: 8px 0; }
.lesson-body :deep(.prose h4) { font-size: 13.5px; margin: 20px 0 8px; display: flex; align-items: center; gap: 7px; }
.lesson-body :deep(.prose h4::before) { content: ''; width: 3px; height: 14px; background: var(--el-color-primary); border-radius: 2px; }
.lesson-body :deep(.prose table) { border-collapse: collapse; width: 100%; font-size: 12.5px; margin: 10px 0; }
.lesson-body :deep(.prose th) { background: #fafbfc; text-align: left; font-weight: 500; padding: 8px 12px; border: 1px solid var(--el-border-color-lighter); }
.lesson-body :deep(.prose td) { padding: 8px 12px; border: 1px solid var(--el-border-color-lighter); }
.lesson-body :deep(.prose pre) { background: #101a3e; color: #d6defa; border-radius: 8px; padding: 14px 16px; font-family: Consolas, monospace; font-size: 12px; line-height: 1.8; overflow: auto; white-space: pre; margin: 10px 0; }
.lesson-body :deep(.prose .mono) { font-family: Consolas, monospace; font-size: 12px; background: #f5f6f8; border-radius: 4px; padding: 1px 5px; }
.lesson-body :deep(.prose .term) { border-bottom: 1px dashed #8f959e; cursor: help; position: relative; }
.lesson-body :deep(.prose .term:hover::after) { content: attr(title); position: absolute; left: 50%; bottom: calc(100% + 8px); transform: translateX(-50%); background: #1f2329; color: #fff; font-size: 12px; line-height: 1.6; padding: 8px 12px; border-radius: 8px; width: 300px; white-space: normal; z-index: 9; font-weight: 400; box-shadow: 0 8px 24px rgba(31, 35, 41, 0.18); text-align: left; }
/* DN 拆解组件（v-html 注入，必须整链 :deep；seg-dn 需 inline-flex 才能让标签竖排在徽章下） */
.lesson-body :deep(.dn-breakdown) { display: flex; align-items: center; gap: 5px; flex-wrap: wrap; font-family: Consolas, monospace; font-size: 12.5px; background: #fafbfc; border: 1px solid var(--el-border-color); border-radius: 8px; padding: 12px 14px; margin: 10px 0; }
.lesson-body :deep(.dn-breakdown .seg-dn) { display: inline-flex; flex-direction: column; align-items: center; gap: 2px; background: #f0f5ff; color: var(--el-color-primary); border-radius: 6px; padding: 3px 9px; }
.lesson-body :deep(.dn-breakdown .seg-dn b) { font-weight: 600; }
.lesson-body :deep(.dn-breakdown .seg-dn i) { font-style: normal; font-size: 10px; color: #8f959e; font-family: 'PingFang SC', 'Microsoft YaHei', sans-serif; white-space: nowrap; }
.lesson-body :deep(.dn-breakdown .sep) { color: #8f959e; align-self: center; }
.lesson-body :deep(.where) { background: #f0f5ff; border-radius: 8px; padding: 12px 14px; font-size: 12.5px; margin-top: 16px; color: #3147a3; }
.lesson-body :deep(.where b) { display: flex; align-items: center; gap: 6px; margin-bottom: 4px; }
.lesson-body :deep(.faq-list details) { border: 1px solid var(--el-border-color-lighter); border-radius: 8px; overflow: hidden; }
.lesson-body :deep(.faq-list details + details) { margin-top: 8px; }
.lesson-body :deep(.faq-list summary) { cursor: pointer; padding: 11px 14px; font-size: 13px; font-weight: 500; display: flex; align-items: center; gap: 8px; list-style: none; user-select: none; }
.lesson-body :deep(.faq-list summary::-webkit-details-marker) { display: none; }
.lesson-body :deep(.faq-list summary::after) { content: '+'; margin-left: auto; color: #8f959e; font-size: 15px; }
.lesson-body :deep(.faq-list details[open] summary::after) { content: '–'; }
.lesson-body :deep(.faq-list details[open] summary) { background: #fafbfc; color: var(--el-color-primary); }
.lesson-body :deep(.faq-list .fa) { padding: 4px 14px 14px; font-size: 12.5px; color: #646a73; line-height: 1.8; }
</style>
