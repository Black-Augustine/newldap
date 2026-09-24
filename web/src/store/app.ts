// 全局 UI 共享状态（跨布局/页面：树选择、模式、树刷新信号）
import { reactive, ref } from 'vue'
import type { TreeNode } from '../api'

export const app = reactive({
  selectedOU: null as TreeNode | null, // 当前选中部门（树 → 人员页）
  expertMode: false,                    // 简单/专家模式（人员抽屉默认页签）
  treeVersion: 0,                       // ++ 触发树重挂载
  peopleVersion: 0,                     // ++ 通知人员列表刷新（树上行内新建等跨组件场景）
  personCreateOpen: false,              // 全局「新建人员」对话框（树行按钮/人员页共用）
  personCreateOU: '',                   // 新建人员默认部门 DN
})

export function bumpTree() { app.treeVersion++ }
export function bumpPeople() { app.peopleVersion++ }
export function setSelected(n: TreeNode | null) { app.selectedOU = n }
export function openPersonCreate(ou = '') {
  app.personCreateOU = ou
  app.personCreateOpen = true
}

export const searchQuery = ref('') // 顶栏全局搜索

// 全局条目抽屉（任意页面可开：树节点/搜索结果/表格行）
export const entryDrawerDN = ref('')
export function openEntry(dn: string) { entryDrawerDN.value = dn } // 顶栏全局搜索（回车 → 人员页搜索模式）

// 新手聚光灯导览开关（帮助中心/概览触发，AdminLayout 挂载）
export const tourOpen = ref(false)

// 系统设置策略（AdminLayout 启动时载入；系统设置页保存后刷新）
export const policy = ref<null | { passwordMinLen: number; passwordComplexity: boolean; uidMinLen: number; uidMaxLen: number; emailAutofill: boolean }>(null)
export function setPolicy(p: typeof policy.value) { policy.value = p }
// 当前 Base DN 的域名形式（dc=example,dc=cn → example.cn），邮箱自动拼接用
export const baseDomain = ref('')

// 载入一次策略（AdminLayout 启动时；新建/编辑人员组件读取 policy.value）
let policyLoaded = false
export async function loadPolicyOnce() {
  if (policyLoaded) return
  policyLoaded = true
  try {
    const { getPolicy } = await import('../api')
    policy.value = await getPolicy()
  } catch { /* 忽略：组件各自兜底默认值 */ }
}
