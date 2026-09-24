// 对接指引：产品模板常量（帮助中心 · R12）。
// 每个模板决定信息卡的字段取舍与文案侧重；全部为纯前端数据，无后端状态。

export interface Tpl {
  key: string
  icon: string // lucide 图标名（组件里映射）
  name: string
  full: string
  desc: string
  acct: string // 建议的只读账号名
  org: boolean // 是否展示组织架构映射
  grp: boolean // 是否展示用户组映射
  tip: string
}

export const TPLS: Tpl[] = [
  {
    key: 'bastion', icon: 'Server', name: '堡垒机', full: 'JumpServer / 齐治等',
    desc: '运维审计系统，按用户与用户组授权资产', acct: 'jumphub', org: true, grp: true,
    tip: 'JumpServer「系统设置 → 认证 → LDAP」逐项对照：LDAP 服务器 / Base DN / 绑定 DN 用下方取值；用户属性映射——用户名=uid、姓名=cn、邮箱=mail；用户树与用户过滤器见「用户映射」一节。资产授权建议用「权限组」。',
  },
  {
    key: 'zero', icon: 'ShieldCheck', name: '零信任 / SDP', full: '深信服 aTrust / SDP 网关',
    desc: '身份为准入依据，组映射权限策略', acct: 'sdp-reader', org: true, grp: true,
    tip: '唯一标识强烈建议用 entryUUID 而非 uid：员工改名、换部门后身份不丢，授权关系保持连续。组授权用「权限组」（groupOfNames），其成员是完整 DN，与零信任的"身份组"模型天然对齐。',
  },
  {
    key: 'vpn', icon: 'Network', name: 'VPN / 网络设备', full: '防火墙 / 交换机 / SSL VPN',
    desc: '设备换 LDAP 认证，通常仅简单 bind + 单个过滤器', acct: 'vpn-auth', org: false, grp: false,
    tip: '多数网络设备只支持「服务器 + Base DN + Bind DN + 一个用户过滤器」，不支持组织架构与组映射——只填「连接」和「用户映射」两节即可。部分老设备不支持 LDAPS，需在目录侧放开明文 389 端口（限内网可达）。',
  },
  {
    key: 'sso', icon: 'Fingerprint', name: '单点登录', full: 'OIDC / SAML 门户的 LDAP 认证源',
    desc: '门户登录的身份源', acct: 'sso-bind', org: false, grp: true,
    tip: '两种接法：「LDAP 直连认证」每次登录实时向目录验证（推荐，密码改后即时生效）；「全量同步到本地」门户首启快，但从此两套账号要人工对齐。选直连时无需组织架构映射。',
  },
  {
    key: 'generic', icon: 'Plug', name: '通用 LDAP 客户端', full: 'Nginx / Jenkins / GitLab / Confluence…',
    desc: '任何支持 LDAP 认证的软件', acct: 'app-readonly', org: true, grp: true,
    tip: '这一档是全字段对照表：对方配置表单里出现的每个 LDAP 字段，下方信息卡都有对应取值。高级参数（超时、分页大小、检索深度）保持对方默认即可，无需修改。',
  },
]

export const USER_FILTER = '(objectClass=inetOrgPerson)'
export const OU_FILTER = '(objectClass=organizationalUnit)'
export const GROUP_FILTER = '(|(objectClass=groupOfNames)(objectClass=posixGroup))'
