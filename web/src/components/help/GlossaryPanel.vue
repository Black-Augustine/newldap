<template>
  <div class="nl-card">
    <div class="nl-card-head">
      <span class="t"><el-icon><BookOpen /></el-icon>术语词典</span>
      <div style="display: flex; gap: 10px; align-items: center">
        <el-input v-model="q" placeholder="搜索属性 / 中文名，如 mail" clearable style="width: 230px">
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-radio-group v-model="kind" size="small">
          <el-radio-button value="attr">属性（{{ attrRows.length }}）</el-radio-button>
          <el-radio-button value="oc">对象类（{{ ocRows.length }}）</el-radio-button>
        </el-radio-group>
      </div>
    </div>
    <el-table :data="rows" size="default" style="width: 100%">
      <el-table-column label="名称" width="200">
        <template #default="{ row }"><span class="mono">{{ row.name }}</span></template>
      </el-table-column>
      <el-table-column label="中文" width="150">
        <template #default="{ row }"><b style="font-size: 12.5px">{{ row.cn }}</b></template>
      </el-table-column>
      <el-table-column label="白话说明" min-width="380">
        <template #default="{ row }"><span style="color: #646a73">{{ row.desc }}</span></template>
      </el-table-column>
      <template #empty>
        <div class="nl-empty"><el-empty :image-size="72" :description="`没有匹配「${q}」的词条`" /></div>
      </template>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { BookOpen, Search } from 'lucide-vue-next'
import { attrDictEntries, classDictEntries } from '../../utils/ldapDict'

const q = ref('')
const kind = ref<'attr' | 'oc'>('attr')

// 核心基础词条（人员/组/部门日常字段）+ ldapDict 悬停词典全量，按名称去重
const BASE_ATTRS: Array<[string, string, string]> = [
  ['cn', '姓名 / 通用名', '条目的显示名。人员的姓名、组的组名、角色名都用它'],
  ['sn', '姓', '人员的姓氏，schema 必填，由姓名自动推导（含复姓）'],
  ['uid', '账号', '登录用账号（如 zhangwei），全组织唯一'],
  ['mail', '邮箱', 'RFC 822 格式，对接系统常用作用户邮箱字段'],
  ['mobile', '手机', '移动电话'],
  ['title', '职位', '职务头衔'],
  ['ou', '部门 / 组织单元', '部门条目的标识。多值：第 1 个 = 英文标识（DN 用），第 2 个 = 中文名'],
  ['employeeNumber', '工号', '企业内部编号，导入时用于识别同一人'],
  ['description', '说明', '通用描述字段：部门说明、组说明都在这'],
  ['member', '组成员（DN）', '权限组（groupOfNames）的成员，值为完整 DN'],
  ['gidNumber', '组 ID', '登录组的数字编号，Linux 用；5000-5999 自动分配'],
  ['uidNumber', '用户 ID', 'Linux 登录账号的数字编号'],
  ['homeDirectory', '主目录', 'Linux 登录后的 home 路径'],
  ['loginShell', '登录 Shell', '如 /bin/bash'],
  ['ouOrder', '部门排序', '本部署扩展的部门显示顺序值，目录树按它排序'],
]

const attrRows = computed(() => {
  const map = new Map<string, { name: string; cn: string; desc: string }>()
  for (const [name, cn, desc] of BASE_ATTRS) map.set(name.toLowerCase(), { name, cn, desc })
  for (const [name, v] of attrDictEntries()) {
    const k = name.toLowerCase()
    if (!map.has(k)) map.set(k, { name, cn: v.cn, desc: v.desc })
  }
  return [...map.values()].sort((a, b) => a.name.localeCompare(b.name))
})

const ocRows = computed(() =>
  classDictEntries().map(([name, v]) => ({ name, cn: v.cn, desc: v.desc }))
    .sort((a, b) => a.name.localeCompare(b.name)),
)

const rows = computed(() => {
  const src = kind.value === 'attr' ? attrRows.value : ocRows.value
  const kw = q.value.trim().toLowerCase()
  if (!kw) return src
  return src.filter((r) => (r.name + r.cn + r.desc).toLowerCase().includes(kw))
})
</script>
