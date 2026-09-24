<template>
  <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 14px">
    <div class="nl-card">
      <div class="nl-card-head">
        <span class="t"><el-icon><ListChecks /></el-icon>上手任务清单</span>
        <span class="nl-pill plain">完成状态实时从目录推断</span>
      </div>
      <div class="nl-card-body">
        <el-progress :percentage="doneCount / tasks.length * 100" :stroke-width="6" :show-text="false" style="margin-bottom: 4px" />
        <div class="muted" style="margin: 6px 0 10px">
          已完成 {{ doneCount }} / {{ tasks.length }} · 不存任何"已读"状态，目录里有什么这里就显示什么
        </div>
        <div v-for="(t, i) in tasks" :key="i" class="todo" :class="{ done: t.done }">
          <span class="st" :class="{ done: t.done }">
            <el-icon v-if="t.done" :size="11" color="#fff"><Check /></el-icon>
          </span>
          <div style="flex: 1">
            <div class="tt">{{ t.title }}</div>
            <div class="desc">{{ t.desc }}</div>
            <a
              v-if="t.to && !t.done" class="link" @click="go(t)"
            >{{ t.link }} →</a>
          </div>
        </div>
      </div>
    </div>
    <div style="display: flex; flex-direction: column; gap: 14px">
      <div class="nl-card">
        <div class="nl-card-head">
          <span class="t"><el-icon><Spotlight /></el-icon>界面导览</span>
          <span class="nl-pill plain">5 步 · 随时重看</span>
        </div>
        <div class="nl-card-body">
          <div class="tour-preview">
            <span class="tp-step"><i>1</i>目录树</span><span class="tp-arrow">→</span>
            <span class="tp-step"><i>2</i>新建部门</span><span class="tp-arrow">→</span>
            <span class="tp-step"><i>3</i>组织与人员</span><span class="tp-arrow">→</span>
            <span class="tp-step"><i>4</i>全局搜索</span><span class="tp-arrow">→</span>
            <span class="tp-step"><i>5</i>帮助中心</span>
          </div>
          <div class="muted" style="margin: 10px 0 12px; line-height: 1.7">
            聚光灯高亮 + 一句话白话，认识本产品最核心的 5 个位置。第一次进入管理台时也会自动出现一次。
          </div>
          <el-button type="primary" @click="$emit('tour')"><el-icon><Play /></el-icon>&nbsp;开始导览</el-button>
        </div>
      </div>
      <div class="nl-card">
        <div class="nl-card-head"><span class="t"><el-icon><Compass /></el-icon>帮助中心还有什么</span></div>
        <div class="nl-card-body nav-cards">
          <div class="nc" @click="$emit('go', 'learn')">
            <el-icon :size="18"><GraduationCap /></el-icon>
            <div><b>LDAP 教学</b><p>9 节课全用你目录里的真实数据举例，附可执行查询的过滤器实验器</p></div>
          </div>
          <div class="nc" @click="$emit('go', 'connect')">
            <el-icon :size="18"><PlugZap /></el-icon>
            <div><b>对接指引</b><p>堡垒机 / 零信任接入该填什么：信息卡逐行复制 + 一键创建只读账号</p></div>
          </div>
          <div class="nc" @click="$emit('go', 'glossary')">
            <el-icon :size="18"><BookOpen /></el-icon>
            <div><b>术语词典</b><p>属性 / 对象类的中文名与白话说明，支持搜索</p></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Check, ListChecks, Spotlight, Play, Compass, GraduationCap, PlugZap, BookOpen } from 'lucide-vue-next'
import { setupStatus, searchEntries, peopleList, listGroups } from '../../api'

defineEmits<{ (e: 'tour'): void; (e: 'go', tab: string): void }>()

const router = useRouter()
const st = ref({ configured: false, depts: 0, people: 0, groups: 0 })

const tasks = computed(() => [
  {
    title: '连接目录并完成探测', desc: st.value.configured ? '已连接，健康状态见概览页' : '运行连接向导，30 秒接入',
    done: st.value.configured, to: '/settings', link: '去系统设置',
  },
  {
    title: '搭建组织结构', desc: st.value.depts > 0 ? `目录中已有 ${st.value.depts} 个部门` : '在左侧树新建部门，或用 Excel 一次导入',
    done: st.value.depts > 0, to: '/people', link: '去组织与人员',
  },
  {
    title: '录入首批人员', desc: st.value.people > 0 ? `共 ${st.value.people} 名人员` : '逐个新建或 Excel 批量导入',
    done: st.value.people > 0, to: '/import', link: '去导入',
  },
  {
    title: '创建用户组', desc: st.value.groups > 0 ? `共 ${st.value.groups} 个组（权限组 / 登录组）` : '权限组用于授权（VPN、堡垒机），登录组用于 Linux',
    done: st.value.groups > 0, to: '/groups', link: '去用户组',
  },
  {
    title: '体验搜索与树定位', desc: '顶部搜索框输入姓名，点结果直接打开人员详情',
    done: false, to: '/people', link: '去试一下',
  },
  {
    title: '对接第一个系统', desc: '堡垒机 / 零信任 / VPN 接入本目录该填什么，一键生成对接信息卡',
    done: false, to: '/help?tab=connect', link: '去对接指引',
  },
])
const doneCount = computed(() => tasks.value.filter((t) => t.done).length)

function go(t: { to: string }) {
  const [path, query] = t.to.split('?')
  router.push({ path, query: query ? Object.fromEntries([query.split('=')]) : undefined })
}

onMounted(async () => {
  try {
    const [sp, ou, ppl, gs] = await Promise.all([
      setupStatus(),
      searchEntries({ filter: '(objectClass=organizationalUnit)', attr: ['ou'], limit: 1 }),
      peopleList(),
      listGroups(),
    ])
    st.value.configured = !!sp.configured
    st.value.depts = ou.total
    st.value.people = ppl.total
    st.value.groups = gs.total ?? 0
  } catch { /* 忽略：清单显示为未完成 */ }
})
</script>

<style scoped>
.todo { display: flex; align-items: flex-start; gap: 10px; padding: 9px 0; border-bottom: 1px dashed var(--el-border-color-lighter); font-size: 13px; }
.todo:last-child { border-bottom: 0; }
.todo .st { width: 18px; height: 18px; border-radius: 50%; border: 1.5px solid var(--el-border-color); display: inline-flex; align-items: center; justify-content: center; flex: none; margin-top: 1px; }
.todo .st.done { background: var(--el-color-primary); border-color: var(--el-color-primary); }
.todo .desc { font-size: 12px; color: #8f959e; margin-top: 2px; }
.todo.done .tt { color: #8f959e; text-decoration: line-through; }
.link { color: var(--el-color-primary); cursor: pointer; font-size: 12px; }
.tour-preview { display: flex; align-items: center; gap: 7px; flex-wrap: wrap; }
.tp-step { display: inline-flex; align-items: center; gap: 6px; font-size: 12px; color: #646a73; background: #fafbfc; border: 1px solid var(--el-border-color-lighter); border-radius: 13px; padding: 4px 10px 4px 5px; }
.tp-step i { width: 17px; height: 17px; border-radius: 50%; background: var(--el-color-primary); color: #fff; font-style: normal; font-size: 10.5px; display: inline-flex; align-items: center; justify-content: center; flex: none; }
.tp-arrow { color: #a8abb2; font-size: 12px; }
.nav-cards { display: flex; flex-direction: column; gap: 8px; }
.nc { display: flex; gap: 10px; align-items: flex-start; border: 1px solid var(--el-border-color-lighter); border-radius: 8px; padding: 11px 13px; cursor: pointer; transition: 0.12s; }
.nc:hover { border-color: var(--el-color-primary); background: #f0f5ff; }
.nc .el-icon { color: var(--el-color-primary); margin-top: 2px; }
.nc b { font-size: 13px; display: block; }
.nc p { font-size: 11.5px; color: #8f959e; margin: 3px 0 0; line-height: 1.6; }
@media (max-width: 1100px) { :deep(.grid) { grid-template-columns: 1fr; } }
</style>
