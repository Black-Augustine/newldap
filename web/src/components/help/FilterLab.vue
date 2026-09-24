<template>
  <div class="lab">
    <div class="lab-h">
      <el-input
        v-model="q" class="mono" placeholder="输入 LDAP 过滤器，如 (uid=zhangwei)"
        clearable @keyup.enter="run"
      />
      <el-button type="primary" :loading="loading" @click="run">
        <el-icon><Play /></el-icon>&nbsp;执行查询
      </el-button>
      <span class="muted">对当前目录执行真实子树搜索（只读，最多 {{ LIMIT }} 条）</span>
    </div>
    <div class="lab-chips">
      <span
        v-for="[n, f] in chips" :key="f" class="lab-chip" :class="{ on: q === f }"
        :title="f" @click="q = f; run()"
      >{{ n }}</span>
    </div>
    <div class="lab-out">
      <template v-if="!result && !error">
        <div class="muted" style="padding: 6px 2px">点「执行查询」试试——结果会是真实条目。</div>
      </template>
      <template v-else-if="error">
        <div class="nl-hint dgr">
          <el-icon style="margin-top: 2px"><TriangleAlert /></el-icon>
          <div>
            <b>服务器拒绝：{{ errorKind }}</b><br />
            {{ error }}<br />
            <span class="muted">白话解释：{{ whiteTalk }}。对接系统配置过滤器写错时，看到的就是这种报错——对着错误和白话改，不慌。</span>
          </div>
        </div>
      </template>
      <template v-else-if="result">
        <div class="lab-hit" :class="result.total ? 'ok' : 'warn'">
          <el-icon><component :is="result.total ? CircleCheck : SearchX" /></el-icon>
          <div>
            语法正确 · 命中 <b>{{ result.total }}</b> 个条目<template v-if="result.total > result.entries.length">（仅显示前 {{ result.entries.length }}）</template>
            <template v-if="!result.total">。真实 LDAP 也会这样返回空列表——对接系统"搜不到用户"多半是过滤器或 Base DN 范围的问题，不是连不上</template>
          </div>
        </div>
        <el-table :data="result.entries" size="small" style="width: 100%">
          <el-table-column label="命中条目 DN" min-width="380">
            <template #default="{ row }"><span class="mono" style="font-size: 12px">{{ row.dn }}</span></template>
          </el-table-column>
          <el-table-column label="类型" width="110">
            <template #default="{ row }">
              <span class="nl-pill plain">{{ kindOf(row) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="姓名 / 名称" width="130">
            <template #default="{ row }">{{ row.attrs.cn?.[0] ?? row.attrs.ou?.[0] ?? '—' }}</template>
          </el-table-column>
        </el-table>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Play, TriangleAlert, CircleCheck, SearchX } from 'lucide-vue-next'
import { searchEntries, errText, type SearchResult } from '../../api'

const LIMIT = 50
const q = ref('(objectClass=inetOrgPerson)')
const loading = ref(false)
const result = ref<SearchResult | null>(null)
const error = ref('')
const errorKind = ref('')

const chips: Array<[string, string]> = [
  ['所有人员', '(objectClass=inetOrgPerson)'],
  ['姓张的', '(cn=张*)'],
  ['有邮箱的人', '(&(objectClass=inetOrgPerson)(mail=*))'],
  ['所有部门', '(objectClass=organizationalUnit)'],
  ['所有组', '(|(objectClass=groupOfNames)(objectClass=posixGroup))'],
  ['组合条件', '(&(objectClass=inetOrgPerson)(|(cn=李*)(cn=王*)))'],
]

async function run() {
  loading.value = true
  error.value = ''
  result.value = null
  try {
    result.value = await searchEntries({
      filter: q.value.trim(),
      attr: ['cn', 'uid', 'ou', 'objectClass'],
      limit: LIMIT,
    })
  } catch (e) {
    error.value = errText(e)
    const msg = error.value.toLowerCase()
    if (msg.includes('201') || msg.includes('filter compile')) errorKind.value = '过滤器编译错误（201/21 Bad search filter）'
    else if (msg.includes('21') || msg.includes('bad search filter')) errorKind.value = '错误 21（Bad search filter）'
    else if (msg.includes('连接') || msg.includes('timeout')) errorKind.value = '网络/连接错误'
    else errorKind.value = '查询被拒绝'
  } finally {
    loading.value = false
  }
}

const whiteTalk = ref('条件必须包在括号里、属性名后要有等号、& 和 | 至少包含一个子条件')
function kindOf(row: unknown): string {
  const oc = (((row as { attrs?: Record<string, string[]> }).attrs ?? {}).objectClass ?? []).map((s) => s.toLowerCase())
  if (oc.includes('inetorgperson')) return '人员'
  if (oc.includes('organizationalunit')) return '部门'
  if (oc.includes('groupofnames')) return '权限组'
  if (oc.includes('posixgroup')) return '登录组'
  if (oc.includes('organizationalrole')) return '服务账号'
  return '条目'
}

run()
</script>

<style scoped>
.lab { border: 1px solid var(--el-border-color); border-radius: 8px; background: #fff; margin-top: 14px; }
.lab-h { display: flex; gap: 8px; padding: 12px 14px; border-bottom: 1px solid var(--el-border-color-lighter); flex-wrap: wrap; align-items: center; }
.lab-h .el-input { width: 340px; }
.lab-chips { display: flex; gap: 6px; flex-wrap: wrap; padding: 10px 14px; border-bottom: 1px solid var(--el-border-color-lighter); }
.lab-chip { font-size: 11.5px; font-family: Consolas, monospace; background: #fafbfc; border: 1px solid var(--el-border-color-lighter); border-radius: 12px; padding: 3px 10px; cursor: pointer; color: #646a73; transition: 0.12s; }
.lab-chip:hover { border-color: var(--el-color-primary); color: var(--el-color-primary); }
.lab-chip.on { border-color: var(--el-color-primary); color: var(--el-color-primary); background: #f0f5ff; }
.lab-out { padding: 12px 14px; font-size: 12.5px; }
.lab-hit { display: flex; gap: 8px; align-items: center; margin-bottom: 10px; font-size: 12.5px; }
.lab-hit.ok { color: #237a1b; }
.lab-hit.warn { color: #a24e08; }
</style>
