<template>
  <div style="max-width: 1280px; margin: 0 auto">
    <div class="nl-page-head">
      <div>
        <h2>概览</h2>
        <div class="crumb">{{ me?.baseDN }} · 数据实时来自目录</div>
      </div>
      <div class="ops">
        <el-button @click="refresh"><el-icon><RefreshCw /></el-icon>&nbsp;刷新统计</el-button>
      </div>
    </div>

    <div class="nl-kpis">
      <div class="nl-card nl-kpi">
        <span class="ico"><el-icon :size="20"><Users /></el-icon></span>
        <div><div class="v">{{ stats.people }}</div><div class="k">人员总数</div></div>
      </div>
      <div class="nl-card nl-kpi">
        <span class="ico"><el-icon :size="20"><FolderTree /></el-icon></span>
        <div><div class="v">{{ stats.depts }}</div><div class="k">部门</div></div>
      </div>
      <div class="nl-card nl-kpi">
        <span class="ico"><el-icon :size="20"><UsersRound /></el-icon></span>
        <div><div class="v">{{ stats.groups }}</div><div class="k">用户组</div></div>
      </div>
      <div class="nl-card nl-kpi">
        <span class="ico"><el-icon :size="20"><ClipboardList /></el-icon></span>
        <div><div class="v">{{ stats.today }}</div><div class="k">今日变更（含 LAM 侧）</div></div>
      </div>
    </div>

    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 14px">
      <div class="nl-card">
        <div class="nl-card-head"><span class="t"><el-icon><ListChecks /></el-icon>快速开始</span><span class="nl-pill plain">新手指引 · {{ doneCount }} / {{ todos.length }}</span></div>
        <div class="nl-card-body">
          <div v-for="(t, i) in todos" :key="i" class="todo" :class="{ done: t.done }">
            <span class="st" :class="{ done: t.done }"><el-icon v-if="t.done" :size="11"><Check /></el-icon></span>
            <div>
              <div class="tt">{{ t.title }}</div>
              <div class="desc">{{ t.desc }}</div>
              <router-link v-if="t.to && !t.done" :to="t.to" class="link" style="font-size: 12px">{{ t.link }} →</router-link>
            </div>
          </div>
        </div>
      </div>
      <div class="nl-card">
        <div class="nl-card-head">
          <span class="t"><el-icon><ShieldCheck /></el-icon>目录状态</span>
          <span class="nl-pill ok"><span class="dot" />{{ health?.ldap === 'up' ? '健康' : '离线' }}</span>
        </div>
        <div class="nl-card-body">
          <div class="kv">
            <span class="k">连接</span><span class="v mono">{{ profile?.profile?.url ?? '—' }}</span>
            <span class="k">Base DN</span><span class="v mono">{{ me?.baseDN }}</span>
            <span class="k">模式</span><span class="v">{{ health?.mock ? '内置演示目录（--mock）' : '外部目录连接' }}</span>
            <span class="k">凭据存储</span><span class="v">{{ profile?.secretKeySet ? 'AES-GCM 加密保存' : '未设置加密密钥（明文/演示）' }}</span>
            <span class="k">编辑保护</span><span class="v">entryCSN 冲突检测已启用</span>
            <span class="k">并存</span><span class="v">与 LAM 等标准客户端实时互见</span>
          </div>
          <div class="nl-hint info" style="margin-top: 12px">
            <el-icon style="margin-top: 2px"><Info /></el-icon>
            <div>本工具不持有任何业务数据库：目录里是什么，这里就是什么。所有修改都是标准 LDAP 操作，其他客户端（LAM / ldapmodify）实时可见。</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  Check, ClipboardList, FolderTree, Info, ListChecks, RefreshCw,
  ShieldCheck, Users, UsersRound,
} from 'lucide-vue-next'
import { errText, healthz, listAudit, listGroups, me as apiMe, searchEntries, setupStatus, type Me, type SetupStatus } from '../api'

const me = ref<Me | null>(null)
const profile = ref<SetupStatus | null>(null)
const health = ref<any>(null)
const stats = ref({ people: 0, depts: 0, groups: 0, today: 0 })

const todos = computed(() => [
  { title: '连接目录并完成探测', desc: profile.value?.configured ? `已连接 ${profile.value.profile?.url ?? ''}` : '未连接，请先运行连接向导', to: '/settings', link: '去连接设置', done: !!profile.value?.configured },
  { title: '搭建组织结构', desc: stats.value.depts > 0 ? `目录中已有 ${stats.value.depts} 个部门` : '在「组织与人员」左侧树中新建部门，或用 Excel 一次导入', to: '/people', link: '去组织与人员', done: stats.value.depts > 0 },
  { title: '录入首批人员', desc: stats.value.people > 0 ? `共 ${stats.value.people} 名人员` : '下载 Excel 模板 → 填写 → 预览计划 → 执行', to: '/import', link: '去导入', done: stats.value.people > 0 },
  { title: '创建用户组', desc: stats.value.groups > 0 ? `共 ${stats.value.groups} 个组 · 权限组用于授权场景` : '权限组用于授权（VPN / 堡垒机），登录组用于 Linux 登录', to: '/groups', link: '去用户组', done: stats.value.groups > 0 },
  { title: '把改密地址发给员工', desc: `https://<本服务地址>/password · 员工自助改密，无需管理员`, to: '/password', link: '预览页面', done: false },
  { title: '对接第一个系统', desc: '堡垒机 / 零信任 / VPN 接入本目录该填什么，一键生成对接信息卡', to: '/help?tab=connect', link: '去对接指引', done: false },
])
const doneCount = computed(() => todos.value.filter((t) => t.done).length)

async function refresh() {
  try {
    const [p, ou, groups, a] = await Promise.all([
      searchEntries({ filter: '(objectClass=inetOrgPerson)', attr: ['cn'], limit: 1 }),
      searchEntries({ filter: '(objectClass=organizationalUnit)', attr: ['ou'], limit: 1 }),
      listGroups(),
      listAudit(200),
    ])
    stats.value.people = p.total
    stats.value.depts = ou.total
    stats.value.groups = groups.total ?? 0
    // ts 为 UTC ISO 串；按浏览器本地时区判断"今天"（startsWith 数字日期永远匹配不上）
    const fmt = (d: Date) => d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0')
    const today = fmt(new Date())
    stats.value.today = (a.events ?? []).filter((e) => {
      const d = e.ts ? new Date(e.ts) : null
      return !!d && !isNaN(d.getTime()) && fmt(d) === today
    }).length
  } catch (e) {
    stats.value.today = 0
  }
}

onMounted(async () => {
  me.value = await apiMe()
  profile.value = await setupStatus()
  health.value = await healthz()
  refresh()
})
</script>

<style scoped>
.todo { display: flex; align-items: flex-start; gap: 10px; padding: 9px 0; border-bottom: 1px dashed var(--el-border-color-lighter); font-size: 13px; }
.todo:last-child { border-bottom: 0; }
.todo .st { width: 18px; height: 18px; border-radius: 50%; border: 1.5px solid var(--el-border-color); display: inline-flex; align-items: center; justify-content: center; flex: none; margin-top: 1px; color: #fff; }
.todo .st.done { background: var(--el-color-primary); border-color: var(--el-color-primary); }
.todo .desc { font-size: 12px; color: #8f959e; margin-top: 2px; }
.todo.done .tt { color: #8f959e; }
.link { color: var(--el-color-primary); text-decoration: none; }
.kv { display: grid; grid-template-columns: auto 1fr; gap: 8px 18px; font-size: 13px; }
.kv .k { color: #8f959e; white-space: nowrap; display: flex; align-items: center; gap: 6px; }
.kv .v { text-align: right; font-size: 12.5px; }
</style>
