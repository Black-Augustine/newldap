import axios from 'axios'

// paramsSerializer.indexes: null 让数组参数序列化为 attr=a&attr=b
// （默认的 attr[]=a 会被后端忽略，导致返回全部属性）
export const http = axios.create({
  withCredentials: true,
  paramsSerializer: { indexes: null },
})

export interface TreeNode {
  dn: string
  rdn: string
  name: string
  objectClass: string[]
  hasChildren: boolean
  childCount: number
  children?: TreeNode[]
}

export interface EntryDetail {
  dn: string
  attrs: Record<string, string[]>
  entryCSN: string
}

export interface Me { bindDN: string; baseDN: string }

// ---------- 连接向导（M1：R1.3/R1.4/R1.5） ----------

export interface SetupStatus {
  configured: boolean
  mock: boolean
  source?: 'env' | 'file' | 'default' | 'mock'
  secretKeySet: boolean
  profile?: { name: string; url: string; baseDN: string; hasBindDN: boolean }
}

export interface ProbeResult {
  ok: boolean
  error?: string
  namingContexts?: string[]
  vendorName?: string
  paging?: boolean
  passwordModify?: boolean
  bind?: 'ok' | 'fail'
  bindError?: string
  base?: { dn: string; exists: boolean; hasChildren: boolean }
}

export async function setupStatus(): Promise<SetupStatus> {
  const { data } = await http.get<SetupStatus>('/api/v1/setup/status')
  return data
}

export async function probe(req: {
  url: string; startTLS?: boolean; insecureSkipVerify?: boolean
  bindDN?: string; bindPassword?: string; baseDN?: string
}): Promise<ProbeResult> {
  const { data } = await http.post<ProbeResult>('/api/v1/probe', req)
  return data
}

export async function saveProfile(req: {
  name?: string; url: string; startTLS?: boolean; insecureSkipVerify?: boolean
  bindDN?: string; bindPassword?: string; baseDN: string; rememberPassword?: boolean
}): Promise<{ saved: boolean; plaintextSaved: boolean }> {
  const { data } = await http.post('/api/v1/setup/profile', req)
  return data
}

// ---------- 目录操作 ----------

export async function login(bindDN: string, password: string) {
  await http.post('/api/v1/auth/login', { bindDN, password })
}

export async function logout() {
  await http.post('/api/v1/auth/logout')
}

export async function me(): Promise<Me> {
  const { data } = await http.get<Me>('/api/v1/me')
  return data
}

export async function tree(base: string, opts?: { ouOnly?: boolean; containers?: boolean }): Promise<TreeNode[]> {
  const { data } = await http.get<{ children: TreeNode[] }>('/api/v1/tree', {
    params: {
      base,
      ouOnly: opts?.ouOnly ? 1 : undefined,
      containers: opts?.containers ? 1 : undefined,
    },
  })
  return data.children ?? []
}

/** 整棵 OU 嵌套树（部门选择器预载；一次请求，预设值可正常显示） */
export async function ouTree(): Promise<TreeNode[]> {
  const { data } = await http.get<{ children: TreeNode[] }>('/api/v1/tree', {
    params: { ouOnly: 1, deep: 1 },
  })
  return data.children ?? []
}

export async function getEntry(dn: string): Promise<EntryDetail> {
  const { data } = await http.get<EntryDetail>('/api/v1/entry', { params: { dn } })
  return data
}

export async function createEntry(dn: string, objectClass: string[], attrs: Record<string, string[]>) {
  await http.post('/api/v1/entry', { dn, objectClass, attrs })
}

export async function updateEntry(dn: string, expectedCSN: string, changes: Array<{ op: string; attr: string; vals: string[] }>) {
  const { data } = await http.put<{ entryCSN: string }>('/api/v1/entry', { dn, expectedCSN, changes })
  return data.entryCSN
}

export interface MovePreview {
  oldDN: string
  newDN: string
  rename: boolean
  moved: number
}

export async function movePreview(dn: string, newRDN?: string, newParent?: string): Promise<MovePreview> {
  const { data } = await http.post<MovePreview>('/api/v1/entry/move-preview', { dn, newRDN, newParent })
  return data
}

export async function moveEntry(dn: string, newRDN?: string, newParent?: string): Promise<string> {
  const { data } = await http.post<{ dn: string }>('/api/v1/entry/move', { dn, newRDN, newParent })
  return data.dn
}

export interface DeletePreview {
  dn: string
  children: number
  subtree: number
  sample: string[]
  empty: boolean
}

export async function deletePreview(dn: string): Promise<DeletePreview> {
  const { data } = await http.get<DeletePreview>('/api/v1/entry/delete-preview', { params: { dn } })
  return data
}

export async function deleteEntry(dn: string, cascade: boolean) {
  await http.post('/api/v1/entry/delete', { dn, cascade })
}

export interface SearchRow { dn: string; attrs: Record<string, string[]> }
export interface SearchResult { total: number; offset: number; limit: number; entries: SearchRow[] }

export async function searchEntries(params: {
  base?: string; q?: string; filter?: string; attr?: string[]; limit?: number; offset?: number
}): Promise<SearchResult> {
  const { data } = await http.get<SearchResult>('/api/v1/search', { params })
  return data
}

export async function searchPeople(base: string) {
  const res = await searchEntries({
    base, filter: '(objectClass=inetOrgPerson)',
    attr: ['cn', 'uid', 'mail', 'mobile', 'employeeNumber', 'title'], limit: 200,
  })
  return res.entries
}

export function errText(e: unknown): string {
  if (axios.isAxiosError(e)) {
    const msg = e.response?.data?.error
    if (msg) return String(msg)
    return e.message
  }
  return String(e)
}

/** 把 DN 拆成 [rdn1, rdn2, ...]（去掉值中转义的逗号暂不处理——管理台输入的 DN 常规形态即可）。 */
export function dnParts(dn: string): string[] {
  return dn.split(',').map((s) => s.trim()).filter(Boolean)
}

/** 由 RDN（如 ou=tech）取属性名。 */
export function rdnAttr(rdn: string): string {
  const i = rdn.indexOf('=')
  return i > 0 ? rdn.slice(0, i) : ''
}

/** 由 RDN 取值。 */
export function rdnValue(rdn: string): string {
  const i = rdn.indexOf('=')
  return i > 0 ? rdn.slice(i + 1) : rdn
}

// ---------- M2：schema 表单 / 人员 / 用户组 / LDIF ----------

export interface FormField {
  name: string
  label: string
  required: boolean
  multi: boolean
  control: string // text|textarea|number|email|tel|datetime|boolean|binary
  syntax?: string
  desc?: string
  readOnly?: boolean
}

export interface FormModel {
  structuralClass: string
  classes: string[]
  fields: FormField[]
}

export async function schemaForm(className: string): Promise<FormModel> {
  const { data } = await http.get<FormModel>('/api/v1/schema/form', { params: { class: className } })
  return data
}

export interface GroupRef { dn: string; name: string; type: string }

export interface PersonDetail extends EntryDetail {
  groups: GroupRef[]
  disabled: boolean
}

export async function personDetail(dn: string): Promise<PersonDetail> {
  const { data } = await http.get<PersonDetail>('/api/v1/people', { params: { dn } })
  return data
}

export async function createPerson(req: {
  cn: string; uid: string; ou: string; sn?: string
  mail?: string; mobile?: string; employeeNumber?: string; title?: string; initialPassword?: string
}): Promise<{ dn: string; initialPassword: string }> {
  const { data } = await http.post('/api/v1/people', req)
  return data
}

export interface PeopleListItem {
  dn: string
  attrs: Record<string, string[]>
  disabled: boolean
}

export async function peopleList(base?: string): Promise<{ people: PeopleListItem[]; total: number }> {
  const { data } = await http.get('/api/v1/people/list', { params: { base } })
  return data
}

export async function uidSuggest(cn: string, ou?: string): Promise<{ uid: string }> {
  const { data } = await http.get('/api/v1/people/uid-suggest', { params: { cn, ou } })
  return data
}

export async function resetPassword(dn: string, password?: string): Promise<{ password: string }> {
  const { data } = await http.post('/api/v1/people/password', { dn, password: password ?? '' })
  return data
}

export async function disablePerson(dn: string) {
  await http.post('/api/v1/people/disable', { dn })
}

export async function enablePerson(dn: string) {
  await http.post('/api/v1/people/enable', { dn })
}

export async function setDept(dn: string, newParent: string): Promise<{ dn: string }> {
  const { data } = await http.post('/api/v1/people/dept', { dn, newParent })
  return data
}

export interface Group {
  dn: string
  name: string
  scenario: string
  objectClass: string[]
  description?: string
  gidNumber?: string
  members: string[]
  memberCount: number
}

export async function listGroups(): Promise<{ groups: Group[]; total: number }> {
  const { data } = await http.get('/api/v1/groups')
  return data
}

export async function createGroup(req: {
  name: string; scenario: string; description?: string; groupOU?: string
  members?: string[]; memberUids?: string[]; gidNumber?: number
}): Promise<{ dn: string }> {
  const { data } = await http.post('/api/v1/groups', req)
  return data
}

export async function updateGroupMembers(dn: string, add: string[], remove: string[]) {
  await http.post('/api/v1/groups/members', { dn, changes: { add, remove } })
}

export async function updateGroupDesc(dn: string, description: string) {
  await http.put('/api/v1/groups', { dn, description })
}

export async function ldifCreate(ldif: string): Promise<{ dn: string }> {
  const { data } = await http.post('/api/v1/entry/ldif', { ldif })
  return data
}

// ---------- 布局/概览/审计 ----------

export async function healthz(): Promise<{ status: string; ldap: string; mock?: boolean; detail?: string; pldaURL?: string }> {
  const { data } = await http.get('/healthz')
  return data
}

export interface AuditRow {
  ts: string; actor: string; ip: string; op: string; dn?: string; detail?: string; result: string
}

export async function listAudit(limit = 50): Promise<{ events: AuditRow[] }> {
  const { data } = await http.get('/api/v1/audit', { params: { limit } })
  return data
}

export interface ProbeInfo { vendor?: string; paging?: boolean; ppolicy?: boolean; memberOf?: boolean }

// ---------- M3：导入导出 ----------

export interface ImportPlanItem {
  sheet: string; row: number; kind: 'dept' | 'person'; action: 'create' | 'update' | 'error'
  name: string; key: string; detail: string
}

export interface ImportParseResult {
  file: string
  summary: { create: number; update: number; dept: number; error: number }
  items: ImportPlanItem[]
}

export async function parseImport(file: File): Promise<ImportParseResult> {
  const form = new FormData()
  form.append('file', file)
  const { data } = await http.post<ImportParseResult>('/api/v1/import/parse', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return data
}

export interface ImportReportItem {
  sheet: string; row: number; name: string; action: string; status: 'ok' | 'fail' | 'skip'; detail: string
}

export async function executeImport(items: ImportPlanItem[]): Promise<{
  summary: { ok: number; fail: number; skip: number }; results: ImportReportItem[]; initialPasswords: Array<{ dn: string; uid: string; password: string }>
}> {
  const { data } = await http.post('/api/v1/import/execute', { items })
  return data
}

export function templateURL() { return '/api/v1/import/template' }
export function exportURL(base?: string) {
  return base ? `/api/v1/export/people?base=${encodeURIComponent(base)}` : '/api/v1/export/people'
}

// ---------- M5+：条目 schema 视图（贴合 LDAP 的展示层） ----------

export interface AttrView {
  name: string
  label: string
  values: string[]
  group: 'must' | 'may' | 'operational' | 'unknown'
  syntax?: string
  single: boolean
  readOnly: boolean
}

export interface ClassInfo { name: string; kind: string }
export interface AttrBrief { name: string; label: string; syntax?: string }

export interface EntryView {
  attrs: AttrView[]
  groupsOrder: string[]
  objectClasses: ClassInfo[]
  structural: string
  classesChain: string[]
  availableClasses: ClassInfo[]
  availableAttrs: AttrBrief[]
}

export interface EntryViewResult {
  dn: string
  entryCSN: string
  attrs: Record<string, string[]>
  view: EntryView
}

export async function entryView(dn: string): Promise<EntryViewResult> {
  const { data } = await http.get<EntryViewResult>('/api/v1/entry-view', { params: { dn } })
  return data
}

// ---------- M6：帮助中心 · 对接指引（R12） ----------

export async function createBindAccount(name: string, parent?: string): Promise<{ dn: string; password: string }> {
  const { data } = await http.post('/api/v1/integration/bindacct', { name, parent })
  return data
}

export async function bindTest(dn: string, password: string): Promise<{ ok: boolean; message: string }> {
  const { data } = await http.post('/api/v1/integration/bindtest', { dn, password })
  return data
}

// ---------- M7：系统设置 · 录入与安全策略 ----------

export interface Policy {
  passwordMinLen: number
  passwordComplexity: boolean
  uidMinLen: number
  uidMaxLen: number
  emailAutofill: boolean
}

export async function getPolicy(): Promise<Policy> {
  const { data } = await http.get('/api/v1/policy')
  return data
}

export async function savePolicy(p: Policy): Promise<Policy> {
  const { data } = await http.put('/api/v1/policy', p)
  return data
}

// domainFromBaseDN：dc=example,dc=cn → example.cn（邮箱自动拼接的域名来源）
export function domainFromBaseDN(baseDN: string): string {
  return baseDN.split(',')
    .map(s => s.trim())
    .filter(s => s.toLowerCase().startsWith('dc='))
    .map(s => s.slice(3))
    .join('.')
}

// ---------- M8：目录档案网页侧管理 ----------

export async function changeBaseDN(baseDN: string): Promise<{ baseDN: string }> {
  const { data } = await http.put('/api/v1/profile/basedn', { baseDN })
  return data
}
