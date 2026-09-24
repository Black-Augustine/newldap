<template>
  <div style="max-width: 1280px; margin: 0 auto">
    <div class="nl-page-head">
      <div>
        <h2>帮助中心</h2>
        <div class="crumb">新手引导 · LDAP 教学 · 对接指引 · 术语词典 —— 展示的目录数据均为实时读取</div>
      </div>
      <div class="ops">
        <el-button @click="tourOpen = true"><el-icon><Play /></el-icon>&nbsp;重看界面导览</el-button>
      </div>
    </div>

    <div class="htabs">
      <button v-for="t in TABS" :key="t.key" :class="{ on: tab === t.key }" @click="goTab(t.key)">
        <el-icon :size="15"><component :is="t.icon" /></el-icon>{{ t.label }}
        <span v-if="t.key === 'connect'" class="nl-pill info">核心</span>
      </button>
    </div>
    <OnboardingPanel v-show="tab === 'onboard'" @tour="tourOpen = true" @go="goTab" />
    <LearnPanel v-if="showLearn" v-show="tab === 'learn'" />
    <ConnectPanel v-if="showConnect" v-show="tab === 'connect'" />
    <GlossaryPanel v-if="showGlossary" v-show="tab === 'glossary'" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Play, Map, GraduationCap, PlugZap, BookOpen } from 'lucide-vue-next'
import OnboardingPanel from '../components/help/OnboardingPanel.vue'
import LearnPanel from '../components/help/LearnPanel.vue'
import ConnectPanel from '../components/help/ConnectPanel.vue'
import GlossaryPanel from '../components/help/GlossaryPanel.vue'
import { tourOpen } from '../store/app'

// icon 直接引用导入的 lucide 组件（字符串无法被 <component :is> 解析到局部导入）
const TABS = [
  { key: 'onboard', label: '新手引导', icon: Map },
  { key: 'learn', label: 'LDAP 教学', icon: GraduationCap },
  { key: 'connect', label: '对接指引', icon: PlugZap },
  { key: 'glossary', label: '术语词典', icon: BookOpen },
]

const route = useRoute()
const router = useRouter()

const tab = computed(() => {
  const t = String(route.query.tab || 'onboard')
  return (TABS.some((x) => x.key === t) ? t : 'onboard') as string
})

// 懒挂载：首次进入对应页签才实例化（信息卡/词典各自有初始请求）
const showLearn = ref(false)
const showConnect = ref(false)
const showGlossary = ref(false)
watch(tab, (t) => {
  if (t === 'learn') showLearn.value = true
  if (t === 'connect') showConnect.value = true
  if (t === 'glossary') showGlossary.value = true
}, { immediate: true })

function goTab(t: string) {
  router.replace({ query: t === 'onboard' ? {} : { tab: t } })
}
</script>

<style scoped>
.htabs { display: flex; gap: 2px; border-bottom: 1px solid var(--el-border-color); margin-bottom: 16px; }
.htabs button { display: inline-flex; align-items: center; gap: 7px; border: 0; background: transparent; height: 40px; padding: 0 15px; font-size: 13.5px; color: #646a73; cursor: pointer; border-bottom: 2px solid transparent; margin-bottom: -1px; transition: 0.15s; }
.htabs button:hover { color: #1f2329; }
.htabs button.on { color: var(--el-color-primary); font-weight: 600; border-bottom-color: var(--el-color-primary); }
.htabs .nl-pill { transform: scale(0.86); margin-left: 2px; }
</style>
