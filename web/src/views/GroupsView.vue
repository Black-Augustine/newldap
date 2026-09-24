<template>
  <div style="max-width: 1280px; margin: 0 auto">
    <div class="nl-page-head">
      <div>
        <h2>用户组</h2>
        <div class="crumb">权限组用于业务授权（groupOfNames）· 登录组用于 Linux 主机登录（posixGroup）</div>
      </div>
      <div class="ops">
        <el-button type="primary" @click="createOpen = true"><el-icon><UsersRound /></el-icon>&nbsp;新建用户组</el-button>
      </div>
    </div>

    <div class="nl-card nl-table-wrap">
      <div class="nl-toolbar">
        <el-radio-group v-model="typeFilter">
          <el-radio-button value="">全部</el-radio-button>
          <el-radio-button value="权限组">权限组</el-radio-button>
          <el-radio-button value="登录组">登录组</el-radio-button>
        </el-radio-group>
        <el-input v-model="q" placeholder="搜索组名" clearable style="width: 200px">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <span class="spacer" />
        <span class="muted">共 {{ filtered.length }} 个组</span>
      </div>
      <el-table :data="filtered" v-loading="loading" stripe border>
        <el-table-column label="组名 (cn)" min-width="200" resizable>
          <template #default="{ row }">
            <div style="display: flex; align-items: center; gap: 10px">
              <span class="nl-avatar gray"><el-icon :size="13"><UsersRound /></el-icon></span>
              <div style="min-width: 0">
                <div>{{ row.name }}</div>
                <div class="mono muted dn-line" :title="row.dn">{{ row.dn }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="130" resizable>
          <template #default="{ row }">
            <el-tooltip :content="row.scenario === '登录组'
              ? '登录组（posixGroup）：成员存账号 uid，供 Linux 主机登录，gidNumber 自动分配'
              : '权限组（groupOfNames）：成员存完整 DN，用于业务系统授权（VPN / 堡垒机等）'" placement="top">
              <span class="nl-pill" :class="row.scenario === '登录组' ? 'plain' : 'info'">
                <el-icon :size="12" style="margin-right: 3px"><component :is="row.scenario === '登录组' ? Server : Shield" /></el-icon>{{ row.scenario }}
              </span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="说明 (description)" min-width="220" resizable>
          <template #default="{ row }"><span style="color: #646a73">{{ row.description || '—' }}</span></template>
        </el-table-column>
        <el-table-column label="组ID (gidNumber)" width="120" resizable>
          <template #default="{ row }"><span class="mono">{{ row.gidNumber || '—' }}</span></template>
        </el-table-column>
        <el-table-column label="成员数" width="90" resizable>
          <template #default="{ row }"><b>{{ row.memberCount }}</b> <span class="muted">名</span></template>
        </el-table-column>
        <el-table-column label="操作" width="170">
          <template #default="{ row }">
            <el-button size="small" link type="primary" @click="openMembers(row as Group)">成员维护</el-button>
            <el-button size="small" link type="primary" @click="editDesc(row as Group)">改描述</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 新建组 -->
    <el-dialog v-model="createOpen" title="新建用户组" width="560px">
      <el-form label-position="top">
        <el-form-item label="这个组用来做什么？（场景决定目录里的类型，无需懂 objectClass）" required>
          <el-radio-group v-model="nc.scenario">
            <el-radio-button value="权限组">给业务系统授权（如 VPN、代码仓库）</el-radio-button>
            <el-radio-button value="登录组">登录 Linux 主机</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="组名 *" required>
            <el-input v-model="nc.name" placeholder="如：vpn-access" /> </el-form-item></el-col>
          <el-col :span="12"><el-form-item label="说明">
            <el-input v-model="nc.description" /> </el-form-item></el-col>
        </el-row>
        <el-form-item v-if="nc.scenario === '权限组'" label="首名成员 *（目录要求建组时至少一名成员，建组后可继续添加）">
          <el-select v-model="nc.member" filterable remote clearable :remote-method="searchPersons"
            :loading="searching" placeholder="输入姓名或账号搜索，如：张伟 / zhangwei" style="width: 100%">
            <el-option v-for="p in personOptions" :key="p.dn" :value="p.dn" :label="`${p.name}（${p.uid || p.dn}）`">
              <span>{{ p.name }}</span>
              <span class="mono muted" style="float: right; font-size: 11px">{{ p.uid }}</span>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item v-else label="成员账号（uid，可空，逗号分隔）">
          <el-input v-model="nc.memberUids" class="mono" placeholder="zhangwei, lina" />
        </el-form-item>
        <div class="nl-hint info" v-if="nc.scenario === '登录组'">
          <el-icon style="margin-top: 2px"><Info /></el-icon>
          <div>gidNumber 将在 5000-5999 范围内自动分配空闲值。</div>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="createOpen = false">取消</el-button>
        <el-button type="primary" :loading="busy" :disabled="!nc.name || (nc.scenario === '权限组' && !nc.member)" @click="create">创建</el-button>
      </template>
    </el-dialog>

    <!-- 改描述 -->
    <el-dialog v-model="descDlg" :title="descTarget ? `组描述 · ${descTarget.name}` : '组描述'" width="480px">
      <el-input
        v-model="descText" type="textarea" :rows="3" maxlength="200" show-word-limit
        placeholder="一句话说明这个组的用途，如：远程 VPN 接入授权"
      />
      <div class="muted" style="margin-top: 8px; font-size: 11.5px">
        描述写入条目 description 属性，对接系统可作为组备注读取。
      </div>
      <template #footer>
        <el-button @click="descDlg = false">取消</el-button>
        <el-button type="primary" :loading="descSaving" @click="saveDesc">保存</el-button>
      </template>
    </el-dialog>

    <!-- 成员维护抽屉 -->
    <el-drawer v-model="membersOpen" :title="cur ? `成员 · ${cur.name}（${cur.scenario}）` : ''" size="480px">
      <template v-if="cur">
        <div class="nl-hint info" style="margin-bottom: 12px">
          <el-icon style="margin-top: 2px"><Info /></el-icon>
          <div>{{ cur.scenario === '权限组' ? '成员以 DN 形式写入 member 属性；人员详情页可反查所属组。' : '成员以账号(uid) 写入 memberUid。' }}</div>
        </div>
        <div class="member-list">
          <div class="member" v-for="(m, i) in cur.members" :key="m">
            <span class="mono">{{ m }}</span>
            <el-button size="small" text type="danger" @click="removeMember(i)">移除</el-button>
          </div>
          <div v-if="!cur.members.length" class="muted" style="padding: 4px 0">暂无成员</div>
        </div>
        <div class="add-row">
          <el-select v-model="newMember" filterable remote clearable :remote-method="searchPersons"
            :loading="searching" size="small" style="flex: 1"
            :placeholder="cur.scenario === '权限组' ? '搜索姓名/账号添加成员（自动写入 DN）' : '搜索姓名/账号添加成员（自动写入 uid）'">
            <el-option v-for="p in personOptions" :key="p.dn" :value="cur.scenario === '权限组' ? p.dn : p.uid"
              :label="p.name + '（' + p.uid + '）'" />
          </el-select>
          <el-button size="small" type="primary" :disabled="!newMember" @click="addMember">添加</el-button>
        </div>
        <div class="nl-hint warn" style="margin-top: 12px">
          <el-icon style="margin-top: 2px"><TriangleAlert /></el-icon>
          <div>权限组至少保留一名成员（目录的 MUST 约束）；删除最后一名成员会被服务端拒绝。</div>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Info, Search, Server, Shield, TriangleAlert, UsersRound } from 'lucide-vue-next'
import { createGroup, errText, listGroups, searchEntries, updateGroupDesc, updateGroupMembers, type Group } from '../api'
import { openEntry } from '../store/app'

const groups = ref<Group[]>([])
const loading = ref(false)
const busy = ref(false)
const createOpen = ref(false)
const q = ref('')
const typeFilter = ref('')
const nc = reactive({ scenario: '权限组', name: '', description: '', member: '', memberUids: '' })
const membersOpen = ref(false)
const cur = ref<Group | null>(null)
const newMember = ref('')

// 首成员选择器：按姓名/账号远程搜索（替代手输 DN —— 新手最容易在这里失败）
const personOptions = ref<Array<{ dn: string; name: string; uid: string }>>([])
const searching = ref(false)
let searchSeq = 0
async function searchPersons(q: string) {
  const kw = q.trim()
  if (!kw) { personOptions.value = []; return }
  const seq = ++searchSeq
  searching.value = true
  try {
    const res = await searchEntries({
      q: kw, filter: '(objectClass=inetOrgPerson)', attr: ['cn', 'uid'], limit: 20,
    })
    if (seq !== searchSeq) return // 过期响应丢弃
    personOptions.value = res.entries.map((e) => ({
      dn: e.dn, name: e.attrs.cn?.[0] ?? e.dn, uid: e.attrs.uid?.[0] ?? '',
    }))
  } catch { /* 搜索失败保持空列表 */ } finally {
    if (seq === searchSeq) searching.value = false
  }
}

const filtered = computed(() =>
  groups.value.filter((g) => (!typeFilter.value || g.scenario === typeFilter.value) &&
    (!q.value || g.name.toLowerCase().includes(q.value.toLowerCase()))))

async function refresh() {
  loading.value = true
  try {
    const res = await listGroups()
    groups.value = res.groups ?? []
  } catch (e) { ElMessage.error(errText(e)) } finally { loading.value = false }
}
refresh()

async function create() {
  busy.value = true
  try {
    await createGroup({
      name: nc.name, scenario: nc.scenario, description: nc.description,
      members: nc.scenario === '权限组' && nc.member ? [nc.member] : undefined,
      memberUids: nc.scenario === '登录组' && nc.memberUids
        ? nc.memberUids.split(/[,，]/).map((s) => s.trim()).filter(Boolean) : undefined,
    })
    ElMessage.success('用户组已创建')
    createOpen.value = false
    nc.name = ''; nc.description = ''; nc.member = ''; nc.memberUids = ''
    refresh()
  } catch (e) { ElMessage.error(errText(e)) } finally { busy.value = false }
}

function openMembers(g: Group) {
  cur.value = JSON.parse(JSON.stringify(g)); newMember.value = ''
  personOptions.value = []
  membersOpen.value = true
}

async function addMember() {
  if (!cur.value || !newMember.value.trim()) return
  try {
    await updateGroupMembers(cur.value.dn, [newMember.value.trim()], [])
    ElMessage.success('已添加')
    openMembers(await reload(cur.value.dn))
  } catch (e) { ElMessage.error(errText(e)) }
}

async function removeMember(i: number) {
  if (!cur.value) return
  try {
    await updateGroupMembers(cur.value.dn, [], [cur.value.members[i]])
    ElMessage.success('已移除')
    openMembers(await reload(cur.value.dn))
  } catch (e) { ElMessage.error(errText(e)) }
}

async function reload(dn: string): Promise<Group> {
  await refresh()
  return groups.value.find((g) => g.dn === dn)!
}

// 改描述：固定 el-dialog（此前用 ElMessageBox.prompt，弹窗样式在部分环境下渲染异常）
const descDlg = ref(false)
const descTarget = ref<Group | null>(null)
const descText = ref('')
const descSaving = ref(false)

function editDesc(g: Group) {
  descTarget.value = g
  descText.value = g.description ?? ''
  descDlg.value = true
}

async function saveDesc() {
  if (!descTarget.value) return
  descSaving.value = true
  try {
    await updateGroupDesc(descTarget.value.dn, descText.value)
    ElMessage.success('已更新')
    descDlg.value = false
    refresh()
  } catch (e) {
    ElMessage.error('保存描述失败：' + errText(e))
  } finally {
    descSaving.value = false
  }
}
</script>

<style scoped>
.dn-line { font-size: 10.5px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 340px; }
.member-list { max-height: 320px; overflow: auto; }
.member { display: flex; justify-content: space-between; align-items: center; padding: 8px 10px; border: 1px solid var(--el-border-color-lighter); border-radius: 6px; margin-bottom: 6px; }
.member .mono { font-size: 11.5px; word-break: break-all; }
.add-row { display: flex; gap: 8px; margin-top: 10px; }
</style>
