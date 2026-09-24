// 常见 LDAP 属性/对象类的中文说明字典（专家模式展示用）。
// 短标签直接内联显示；长说明通过 ？ 图标悬停展示。

const ATTRS: Record<string, { cn: string; desc: string }> = {
  createTimestamp: { cn: '创建时间', desc: '该条目第一次写入目录的时刻（UTC，只读）' },
  modifyTimestamp: { cn: '最后修改时间', desc: '该条目最近一次被修改的时刻（UTC，只读），任何客户端（含 LAM）改动都会更新' },
  creatorsName: { cn: '创建者', desc: '创建该条目的操作者 DN（只读）' },
  modifiersName: { cn: '最后修改者', desc: '最近一次修改该条目的操作者 DN（只读）' },
  entryCSN: { cn: '变更序号', desc: '条目变更序号（Change Sequence Number）：每次修改都会变化，本工具用它检测并发编辑冲突（他人先改则拒绝保存并提示刷新）' },
  entryUUID: { cn: '全局唯一 ID', desc: '条目的全局唯一标识（UUID），跨系统引用时比 DN 更稳定' },
  entryDN: { cn: '条目 DN', desc: '条目的完整标识路径（操作属性形式返回）' },
  hasSubordinates: { cn: '是否有子条目', desc: 'yes/no：该条目下是否存在子条目' },
  structuralObjectClass: { cn: '结构类', desc: '决定条目类型的结构对象类' },
  subschemaSubentry: { cn: 'Schema 位置', desc: '该条目适用的 schema 子树的 DN' },
  userPassword: { cn: '登录密码', desc: '账号密码（通常以哈希存储，本工具任何接口都不回显）' },
  supportedControl: { cn: '支持的控制', desc: '服务器支持的 LDAP 扩展控制（OID 列表）' },
  supportedExtension: { cn: '支持的扩展', desc: '服务器支持的 LDAP 扩展操作（OID 列表）' },
  audio: { cn: '音频', desc: '人员的音频简介（极少使用）' },
  businessCategory: { cn: '业务类别', desc: '条目所属的业务分类描述' },
  carLicense: { cn: '车牌号', desc: '公司配车的车牌号码' },
  displayName: { cn: '显示名', desc: '界面优先展示的名称（可不同于 cn）' },
  departmentNumber: { cn: '部门编号', desc: '部门代号（与目录树 OU 独立）' },
  initials: { cn: '缩写', desc: '姓名首字母缩写' },
  physicalDeliveryOfficeName: { cn: '办公位置', desc: '办公室/工位位置' },
  roomNumber: { cn: '房间号', desc: '办公室房间号' },
  telephoneNumber: { cn: '办公电话', desc: '座机号码' },
  secretary: { cn: '秘书', desc: '秘书的 DN 引用' },
  labeledURI: { cn: '个人主页', desc: '个人网站/主页 URL' },
  preferredLanguage: { cn: '首选语言', desc: '偏好语言（如 zh-CN）' },
  manager: { cn: '直属上级', desc: '上级的 DN 引用' },
  givenName: { cn: '名', desc: '名字（不含姓）' },
  employeeType: { cn: '员工类型', desc: '用工类型（正式/外包/实习等）' },
  owner: { cn: '负责人', desc: '组/条目负责人的 DN 引用' },
  o: { cn: '组织名', desc: '所属组织/公司名称' },
  l: { cn: '城市', desc: '所在城市（locality）' },
  st: { cn: '省份', desc: '所在省/州' },
  seeAlso: { cn: '参见', desc: '相关条目的 DN 引用' },
  destinationindicator: { cn: '目的地代码', desc: '电报目的地代码（X.500 遗留）' },
  postalAddress: { cn: '邮寄地址', desc: '邮政通信地址' },
  registeredAddress: { cn: '注册地址', desc: '登记注册地址' },
  teletexTerminalIdentifier: { cn: '智能终端标识', desc: 'Teletex 终端标识（极少使用）' },
  x121Address: { cn: 'X.121 地址', desc: 'X.25 网络地址（极少使用）' },
  searchGuide: { cn: '搜索指引', desc: '目录搜索提示信息（极少使用）' },
  postOfficeBox: { cn: '邮箱号', desc: '邮政信箱' },
  postalCode: { cn: '邮编', desc: '邮政编码' },
  street: { cn: '街道', desc: '街道地址' },
  preferredDeliveryMethod: { cn: '投递方式', desc: '偏好的联系投递方式' },
  destinationIndicator: { cn: '目的地代码', desc: '电报目的地代码（极少使用）' },
  telefax: { cn: '传真', desc: '传真号码（X.500 遗留属性）' },
  facsimiletelephonenumber: { cn: '传真号', desc: '办公传真号码' },
  telexnumber: { cn: '电传号', desc: '电报号码（极少使用）' },
  x121address: { cn: 'X.121 网络地址', desc: 'X.25 网络地址（极少使用）' },
  internationalisdnnumber: { cn: 'ISDN 号码', desc: 'ISDN 综合业务数字网号码（极少使用）' },
  homephone: { cn: '家庭电话', desc: '家庭联系电话' },
  homepage: { cn: '个人主页', desc: '个人网站 URL' },
  jpegphoto: { cn: '照片 (JPEG)', desc: 'JPEG 格式证件照（二进制）' },
  photo: { cn: '照片', desc: '照片（二进制）' },
  usercertificate: { cn: '数字证书', desc: 'X.509 用户证书（二进制）' },
  usersmimecertificate: { cn: 'S/MIME 证书', desc: '邮件加密证书（二进制）' },
  uniquemember: { cn: '成员 (唯一)', desc: '组的唯一成员 DN（groupOfUniqueNames）' },
  roleoccupant: { cn: '角色担任者', desc: '担任该角色的人员 DN' },
  dmdname: { cn: 'DMD 名称', desc: '目录管理域名称（极少使用）' },
  memberuid: { cn: '成员账号', desc: '登录组成员的 uid 列表' },
  objectclass: { cn: '对象类', desc: '条目的类型模板列表，决定可用属性' },
  dc: { cn: '域名组件 (dc)', desc: '构成目录域名后缀的组件' },
  shadowlastchange: { cn: '密码最后修改日', desc: '距 1970-01-01 的天数' },
  shadowmax: { cn: '密码最长有效天数', desc: '超过此天数需修改密码' },
  shadowmin: { cn: '密码最短有效天数', desc: '两次改密的最小间隔' },
  shadowexpire: { cn: '账号过期日', desc: '过期后账号不可用（天数为单位）' },
  info: { cn: '备注信息', desc: '自由备注' },
  maillocaladdress: { cn: '本地邮件地址', desc: '邮件路由用的本地地址' },
  mailhost: { cn: '邮件主机', desc: '邮件服务器地址' },
  namingContexts: { cn: '命名上下文', desc: '服务器承载的数据后缀（Base DN 候选）' },
}

const CLASSES: Record<string, { cn: string; desc: string }> = {
  top: { cn: '基类', desc: '所有条目都必须继承的顶层基类，本身不携带业务属性' },
  person: { cn: '人员', desc: '人员基础类：要求 cn（姓名）与 sn（姓氏）' },
  organizationalPerson: { cn: '组织人员', desc: '在 person 之上增加工作单位联系方式（部门地址、电话等）' },
  inetOrgPerson: { cn: '常用人员', desc: '最常用的人员类：uid（账号）、mail、mobile、title、照片等都来自它——本系统新建人员默认使用' },
  organizationalUnit: { cn: '部门', desc: '组织单元（OU）：目录树中的部门/分支节点，可多层嵌套' },
  groupOfNames: { cn: '权限组', desc: '命名成员组：member 属性保存成员 DN，至少需要一名成员（schema MUST），适合业务授权' },
  posixGroup: { cn: '登录组', desc: 'POSIX 组：gidNumber 标识，memberUid 保存成员账号，供 Linux 主机登录使用' },
  posixAccount: { cn: 'POSIX 账号', desc: 'Linux 登录账号：uidNumber/gidNumber/homeDirectory 等，追加到人员条目后可用 SSH 登录' },
  shadowAccount: { cn: '密码策略账号', desc: '密码老化策略：过期时间、锁定、最后修改时间等' },
  simpleSecurityObject: { cn: '简单密码对象', desc: '允许条目直接持有 userPassword（少数简单条目使用）' },
  dcObject: { cn: '域名组件', desc: 'dc= 命名组件，构成目录的域名后缀（如 dc=example,dc=cn）' },
  extensibleObject: { cn: '可扩展对象', desc: '辅助类：允许条目携带 schema 之外的任意属性（慎用）' },
  applicationProcess: { cn: '应用进程', desc: '应用/服务条目' },
  device: { cn: '设备', desc: '设备条目' },
  organizationalRole: { cn: '角色 / 服务账号', desc: '角色类条目：admin、对接只读账号等，可携带密码（配合 simpleSecurityObject），支持改名与改密' },
  organization: { cn: '组织', desc: '代表一个组织/公司的条目（o= 命名）' },
  country: { cn: '国家', desc: '代表国家的条目（c= 命名）' },
  residentialPerson: { cn: '居民人员', desc: 'person 的扩展，增加住址信息' },
  newldapOUExtras: { cn: '部门扩展（本部署自定义）', desc: 'NewLDAP 自定义辅助类：为部门条目追加 ouOrder（拖拽排序）属性' },
  uidObject: { cn: '账号载体', desc: '辅助类：允许条目携带 uid 属性' },
  labeledURIObject: { cn: 'URI 载体', desc: '辅助类：允许条目携带 labeledURI（网页地址）属性' },
  referral: { cn: '引 referrals', desc: 'LDAP 引用条目：指向其他目录服务器' },
  alias: { cn: '别名', desc: '条目别名：指向目录中的另一条目' },
  subentry: { cn: '子条目', desc: '目录子条目：用于承载访问控制策略等' },
  locality: { cn: '地点', desc: '地理位置条目' },
  // ── COSINE / X.500 标准类 ──
  pilotPerson: { cn: 'Pilot 人员', desc: '早期 COSINE 人员类（已被 inetOrgPerson 取代）' },
  account: { cn: '账号', desc: '通用账号条目（uid 命名，比 inetOrgPerson 更简单）' },
  document: { cn: '文档', desc: '代表一份文档的条目' },
  documentSeries: { cn: '文档集', desc: '代表一组相关文档的条目' },
  room: { cn: '房间', desc: '代表房间/会议室的条目' },
  domain: { cn: '域名', desc: '域名条目（dc 命名，与 dcObject 类似）' },
  domainRelatedObject: { cn: '域名关联对象', desc: '标记条目与某个域名相关' },
  RFC822localPart: { cn: '邮箱前缀', desc: '邮箱地址的本地部分（@ 前段）' },
  dNSDomain: { cn: 'DNS 域名', desc: 'DNS 域条目（包含 DNS 相关属性）' },
  friendlyCountry: { cn: '友好国家名', desc: 'country 的别名形式（用可读名称代替 ISO 代码）' },
  pilotOrganization: { cn: 'Pilot 组织', desc: '早期组织类（已被 organization 取代）' },
  pilotDSA: { cn: 'Pilot 目录代理', desc: '早期目录系统代理类' },
  dSA: { cn: '目录系统代理（DSA)', desc: '代表一个目录服务器实例' },
  applicationEntity: { cn: '应用实体', desc: '代表一个 OSI 应用层实体' },
  strongAuthenticationUser: { cn: '强认证用户', desc: '持有证书的强认证用户' },
  userSecurityInformation: { cn: '用户安全信息', desc: '用户的安全属性（证书等）' },
  certificationAuthority: { cn: '证书颁发机构 (CA)', desc: '代表一个 X.509 证书颁发机构' },
  'certificationAuthority-V2': { cn: '证书颁发机构 V2', desc: 'CA 的 V2 版本（增加 CRL 扩展）' },
  cRLDistributionPoint: { cn: 'CRL 分发点', desc: '证书吊销列表（CRL）的分发点' },
  deltaCRL: { cn: '增量 CRL', desc: '增量证书吊销列表' },
  pkiUser: { cn: 'PKI 用户', desc: '持有用户证书的 PKI 用户' },
  pkiCA: { cn: 'PKI 证书机构', desc: 'PKI 证书颁发机构' },
  dmd: { cn: '目录管理域', desc: '目录管理域条目' },
  qualityLabelledData: { cn: '质量标记数据', desc: '带质量标记的数据条目' },

  // ── OpenLDAP 内部配置类（通常不出现在数据条目中）──
  OpenLDAProotDSE: { cn: 'OpenLDAP 根 DSE', desc: '服务器根 DSE 条目（内部）' },
  subschema: { cn: '子 schema', desc: 'Schema 子条目（内部）' },
  dynamicObject: { cn: '动态对象', desc: '支持动态过期的条目（需 overlay）' },
  olcConfig: { cn: 'OpenLDAP 配置', desc: 'OpenLDAP 配置条目（内部）' },
  olcGlobal: { cn: '全局配置', desc: 'OpenLDAP 全局配置（内部）' },
  olcSchemaConfig: { cn: 'Schema 配置', desc: 'OpenLDAP Schema 配置（内部）' },
  olcBackendConfig: { cn: '后端配置', desc: 'OpenLDAP 后端配置（内部）' },
  olcDatabaseConfig: { cn: '数据库配置', desc: 'OpenLDAP 数据库配置（内部）' },
  olcOverlayConfig: { cn: 'Overlay 配置', desc: 'OpenLDAP Overlay 配置（内部）' },
  olcIncludeFile: { cn: '包含文件', desc: 'OpenLDAP 包含文件配置（内部）' },
  olcFrontendConfig: { cn: '前端配置', desc: 'OpenLDAP 前端配置（内部）' },
  olcModuleList: { cn: '模块列表', desc: 'OpenLDAP 模块配置（内部）' },
  olcLdifConfig: { cn: 'LDIF 配置', desc: 'OpenLDAP LDIF 配置（内部）' },
  olcMdbConfig: { cn: 'MDB 配置', desc: 'OpenLDAP MDB 数据库配置（内部）' },
  olcMemberOf: { cn: 'memberOf 配置', desc: 'memberOf Overlay 配置（内部）' },
  olcRefintConfig: { cn: '引用完整性配置', desc: 'refint Overlay 配置（内部）' },

  // ── NIS / POSIX 类 ──
  ipService: { cn: 'IP 服务', desc: '网络服务条目（端口号/协议）' },
  ipProtocol: { cn: 'IP 协议', desc: '网络协议条目' },
  oncRpc: { cn: 'RPC 服务', desc: 'RPC 服务条目' },
  ipHost: { cn: 'IP 主机', desc: '网络主机条目（IP 地址/主机名）' },
  ipNetwork: { cn: 'IP 网络', desc: '网络条目（子网/掩码）' },
  nisNetgroup: { cn: 'NIS 网络组', desc: 'NIS 网络组（主机/用户三元组集合）' },
  nisMap: { cn: 'NIS 映射', desc: 'NIS 映射条目' },
  nisObject: { cn: 'NIS 对象', desc: 'NIS 对象条目' },
  ieee802Device: { cn: 'MAC 地址设备', desc: '带 MAC 地址的设备' },
  bootableDevice: { cn: '可启动设备', desc: '带启动参数的设备' },

  // ── 密码策略 ──
  pwdPolicy: { cn: '密码策略', desc: 'ppolicy 密码策略条目：定义密码复杂度/过期/锁定等规则' },
  pwdPolicyChecker: { cn: '密码策略检查器', desc: '密码质量检查模块配置' },

  // ── Kopano（邮件/协作套件）──
  'kopano-user': { cn: 'Kopano 用户', desc: 'Kopano 邮件/协作套件的用户条目' },
  'kopano-contact': { cn: 'Kopano 联系人', desc: 'Kopano 联系人条目' },
  'kopano-group': { cn: 'Kopano 组', desc: 'Kopano 分发组/安全组' },
  'kopano-company': { cn: 'Kopano 公司', desc: 'Kopano 多租户公司条目' },
  'kopano-server': { cn: 'Kopano 服务器', desc: 'Kopano 后端服务器节点' },
  'kopano-addresslist': { cn: 'Kopano 地址列表', desc: 'Kopano 全局地址簿视图' },
  'kopano-dynamicgroup': { cn: 'Kopano 动态组', desc: 'Kopano 基于过滤器自动维护成员的组' },

  // ── Postfix 邮件 ──
  PostfixBookMailAccount: { cn: 'Postfix 邮件账号', desc: 'Postfix 邮件服务器的用户账号' },
  PostfixBookMailForward: { cn: 'Postfix 邮件转发', desc: 'Postfix 邮件转发规则' },

  // ── Samba（Windows 域控兼容）──
  sambaSamAccount: { cn: 'Samba 域账号', desc: 'Samba/Windows 域控的用户账号（含 Windows 相关属性）' },
  sambaGroupMapping: { cn: 'Samba 组映射', desc: 'Samba/Windows 域控的组映射（POSIX 组↔Windows 组）' },
  sambaTrustPassword: { cn: 'Samba 信任密码', desc: 'Samba 域间信任关系的密码' },
  sambaTrustedDomainPassword: { cn: 'Samba 受信域密码', desc: '受信域的密码条目' },
  sambaDomain: { cn: 'Samba 域', desc: 'Samba 域信息（SID/策略）' },
  sambaUnixIdPool: { cn: 'Samba ID 池', desc: 'Samba 自动分配 uid/gid 的池' },
  sambaIdmapEntry: { cn: 'Samba ID 映射', desc: 'Samba SID↔uid/gid 映射' },
  sambaSidEntry: { cn: 'Samba SID 条目', desc: 'Samba SID 条目' },
  sambaConfig: { cn: 'Samba 配置', desc: 'Samba 全局配置条目' },
  sambaShare: { cn: 'Samba 共享', desc: 'Samba 文件共享定义' },
  sambaConfigOption: { cn: 'Samba 配置项', desc: 'Samba 单个配置选项' },
  sambaTrustedDomain: { cn: 'Samba 受信域', desc: 'Samba 信任的域条目' },

  // ── SSH 公钥 ──
  ldapPublicKey: { cn: 'SSH 公钥', desc: '辅助类：允许条目携带 SSH 公钥（sshPublicKey 属性），用于 SSH 免密登录' },

  // ── 其他 ──
  groupOfUniqueNames: { cn: '唯一成员组', desc: '类似 groupOfNames，但成员用 uniqueMember（DN + 唯一标识）' },
}

export function attrCN(name: string): string | undefined {
  return ATTRS[name.toLowerCase()]?.cn ?? ATTRS[name]?.cn
}

export function attrDesc(name: string): string | undefined {
  const v = ATTRS[name] ?? ATTRS[name.toLowerCase()]
  return v?.desc
}

export function classCN(name: string): string | undefined {
  return CLASSES[name]?.cn ?? CLASSES[name.toLowerCase()]?.cn
}

export function classDesc(name: string): string | undefined {
  const v = CLASSES[name] ?? CLASSES[name.toLowerCase()]
  return v?.desc
}

// 术语词典页（帮助中心）：导出全量词条供列表渲染。
export function attrDictEntries(): Array<[string, { cn: string; desc: string }]> {
  return Object.entries(ATTRS)
}
export function classDictEntries(): Array<[string, { cn: string; desc: string }]> {
  return Object.entries(CLASSES)
}
