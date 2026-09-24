# Changelog

本项目的显著变更记录。版本号在首次正式发布前为开发迭代线。

## [Unreleased] / v1.x 开发线（截至 2026-09-23）

### 修复（第六批补充，同日）：迁移脚本容器化部署健壮性
- 定位改为脚本所在目录（$(dirname "$0")）——root 或任意用户从任何路径执行均可（原先写死 $HOME/newldap-deploy，root 执行即失败）。
- 卷清理改 docker compose down -v：与项目名/目录名解耦（原先硬编码 newldap-deploy_* 卷名，换目录部署即失效）。
- 无参数执行输出友好用法（原先 ${1:?} 带行号前缀难读）；前置检查 compose 文件与 openldap 服务存在，非 compose 部署给出明确指引。
- 第 5 步 phpldapadmin 等可选服务容错启动。服务器验证：语法 / 无参提示 / 错误目录拒绝三项通过，新脚本已放到部署目录。

### 修复（第七批，2026-09-24）：四项
- ElMessageBox/ElMessage 样式：按需引入漏加载函数式 API 的 CSS → 黑底/无遮罩；
  main.ts 手动 import message-box/message 样式（生产验证 6 类弹窗全白底+遮罩正常）。
- CSRF Origin 校验整体移除（内网代理场景频繁误拦；SameSite=Strict Cookie 仍为主防线）。
- dc 整体迁移功能按用户要求完全移除（后端/前端/服务器脚本/文档清理干净）。
- 对象类词典补齐服务器全部 100 个类中文标注（新增 75+ 词条：COSINE/NIS/
  密码策略/Kopano/Postfix/Samba/SSH 公钥等）。

### 移除（2026-09-24）：重命名组织域名（dc）功能
按用户要求整体移除——实际使用中脚本对部署环境要求过高（用户以 root 从非部署目录执行即失败）。保留 Base DN 修改（管理范围切换）。

### 新增（第六批，2026-09-24）：重命名组织域名（dc）· 高危操作
- 系统设置 → 目录连接新增高危卡与对话框：风险告知（全目录 DN 变化/对接系统失效/停机窗口/entryUUID 重新生成/全员重新登录）+ 风险勾选 + **admin 密码二次验证**（服务端真实 bind 校验，错误即拒）。
- OpenLDAP 协议层禁止在线改后缀（实验验证 modrdn 返回 71 affects multiple DSAs），故验证通过后生成**定制迁移脚本**供运维停机执行：备份（slapcat）→ DN+dc 属性重写 → 剔除不可写操作属性 → 清卷换域名重建 → **注入自定义 schema** → 导入 → 同步 NewLDAP 配置卷 → 强制重启全员重登。
- 演练实证：生产 example.cn→test.com→example.cn 全程跑通（过程中发现并修复两处脚本缺陷：根条目 dc 属性未替换、新库缺 schema-newldap.ldif 导致部门条目导入失败），恢复后 29/29 冒烟。
- 后端 POST /api/v1/directory/dc-migration（审计 dc-migration；同域名/非 dc 形式/错误密码分别拒绝）。

### 修复（第五批，2026-09-24）：七项反馈
- 登录页副标题在“使用目录管理员账号登录”与“（bind 登录…）”之间换行。
- PLDA 自动登录彻底重写为**服务端方案**：服务端直接 POST PLDA 登录、会话 Cookie 随 303 下发到浏览器——旧版失败根因是表单缺 submit=Authenticate 字段且依赖浏览器 JS 提交；新版打开链接即登录（curl 直连/经代理/浏览器 fetch 三重验证）。
- 角色账号编辑入口：帮助中心·对接指引“账号已创建”区新增「编辑该账号（改名/改密）」按钮直开条目抽屉。
- 左下角连接安全文案动态化（ldaps://→LDAPS 加密连接；ldap://→明文连接建议改用 LDAPS），消除与 ldap:// 的矛盾显示。
- 移除顶栏 ？ 下拉（与侧栏帮助中心重复）。
- 用户组改描述改用固定 el-dialog（textarea+字数统计），替代样式异常的 MessageBox。
- 用户组表移除操作列 fixed=right（Element 边框表拖列宽时 fixed 列脱开导致右侧空白）。

### 新增（第四批，同日）：六项体验改进
- 登录页副标题与标题间增加间距换行。
- 角色/服务账号条目（organizationalRole：admin、对接只读账号等）支持改名（cn=RDN 重命名，带查重与确认）与修改密码（复用策略校验）；组条目仍引导至用户组页。
- 用户组页：表格加边框（列宽可拖拽）、DN 单行省略悬停查看、表头全双语（组ID (gidNumber)/成员数）、类型徽章带图标与悬停释义。
- 系统设置 → 目录连接新增「组织 Base DN（dc）」管理卡：展示当前值与域名形式，修改对话框保存前实读验证；保存后对所有用户生效且 wizard_saved 标记使其**优先于部署环境变量**（重启不回退，env→file 自动翻转）。
- phpLDAPadmin 自动登录：新增同源反代 /plda/（ReverseProxy，PLDA 相对链接天然适配）+ 桥接页 /api/v1/plda/bridge 用档案管理员凭据自动提交登录表单；侧栏入口改走桥接（含手动兜底按钮与直连链接）。
- 外层路由注册 /plda/（此前仅内层 mux 导致 SPA 兜底截胡）。

### 修复（第三批，同日）：dc 适配——去除 dc=example,dc=cn 写死
- 登录页 Bind DN 默认值/占位符改为跟随实际 Base DN（来自 /setup/status；未配置时通用占位）。
- 建库向导：三套结构模板的树形示意与 Bind DN 占位符跟随探测/填写的 Base DN。
- 自助改密页域名后缀改为完整多级域名（dc=corp,dc=test → @corp.test，原仅取第一个 dc）。
- 帮助中心：ACL 样例兜底、LDAP 教学中 DN 树根/示例邮箱改用真实目录数据注入。
- 验证方式：以伪 Base DN（dc=corp,dc=test）起独立实例，登录页/改密页零 example 残留。

### 新增（第二批，同日）
- **系统设置**（原「连接设置」并入）：安全与录入策略页签——uid 最小/最大长度、密码最小长度与复杂度开关、**工号自动拼邮箱**（只填工号邮箱留空 → 工号@域名，域名取 Base DN dc 部分，手动输入优先），策略存本机配置文件（compose 挂 newldap-data 卷持久化）、热生效、修改有审计；目录连接页签含「切换目录服务器」。
- 服务端强制：建人校验 uid 长度与指定初始密码复杂度，重置密码校验策略（GET/PUT /api/v1/policy）。
- 修复：CSRF 防护误拦——主机比对大小写不敏感、默认端口归一、接受 X-Forwarded-Host 首跳、Origin:null（沙箱 iframe）放行，真正异源仍拒（e2e 六组断言）。
- 修复：登录页新增常驻「切换目录服务器」入口；向导在已配置时可匿名重跑（原 403 造成服务器变更后无法登录的死锁），已配置时向导页展示替换告警（env 来源提示重启以环境变量为准）；setup/status 增加 source 字段。
- 连接向导保存不再丢失策略字段（整份配置原子落盘）；SaveToFile 补 Policy 序列化。

### 新增
- **帮助中心**（侧栏第 7 项）：
  - 新手引导：6 步上手清单（完成态实时从目录推断）+ 5 步自研聚光灯导览（首次登录自动出现）。
  - LDAP 教学：9 节课程全用目录真实数据举例（DN 拆解 / objectClass / 多值属性 / bind 与 ACL / 过滤器 / 两种组 / LDIF / 树设计建议 / 常见误区），内嵌**过滤器实验器**（对真实目录执行查询，语法错误展示服务器真实报错 + 白话）。
  - 对接指引：对外通告地址 → 5 种产品模板（堡垒机 / 零信任 / VPN / SSO / 通用）→ 按对方表单分组的信息卡（逐行复制 / 一键复制）→ 对接专用只读账号（位置可选，默认自动建 ou=services；organizationalRole + simpleSecurityObject 标准类）+ 一次性密码 + 凭据自测（真实 bind，49 错误白话）+ olcAccess 样例 LDIF → 接入验证 3 步与 FAQ。
  - 术语词典：属性 / 对象类中文对照，可搜索。
  - 后端新增 `POST /api/v1/integration/bindacct`、`POST /api/v1/integration/bindtest`（审计不落密码）。
- 目录树：根节点「全部」查看全员、部门拖拽排序（ouOrder 持久化）、OU 中文名（第二个 ou 值）、OU 行内新建人员按钮。
- 人员：uid 全组织唯一 + 拼音自动生成 + 防抖查重、批量移动部门、自定义列与 ▲▼ 排序、列宽拖动、禁用状态真实化（服务端判定 userPassword，不出网）。
- 组：成员远程搜索添加、成员列表限高、组条目「条目信息」视图（防误编辑人员字段）。
- 专家模式条目抽屉：属性四页签（属性 / 对象类 / 操作属性 / LDIF）、对象类可视化编辑（更换主类 / 辅助类分流）、新增属性行、所属组反查。
- 概览「今日变更」本地时区统计；快速开始升级 6 步。
- 生产部署套件：docker-compose.prod.yml 三件套、schema-newldap.ldif、seed.ldif、run-tests.sh（29 项冒烟）。
- CI：vet + 全量测试 / 真实 OpenLDAP 集成 / 前端 typecheck + build / 镜像构建。

### 修复（节选）
- `GET /api/v1/entry` 泄漏 userPassword（P0）：StripSensitive 出口统一剥离 + e2e 回归。
- 更换主类保存无效：pendingMain 在"无修改"早退前消费。
- 今日变更恒 0：dashless 日期串与 ISO 不匹配 + UTC 时区。
- defaultOu prop 大小写陷阱、el-tree-select 懒加载清预设、嵌套 el-dialog 关闭卡死等前端系列修复。
- mock delete 错误码区分（不存在=32）；run-tests.sh 加成员请求结构对齐嵌套 changes。

### 历史（M0–M5）
- M0 技术验证（go-ldap / BER 替身 / 分页宽容）
- M1 连接向导 / 目录树 / 删除保护 / CSRF / 凭据加密
- M2 人员 / 组领域层 / Schema 动态表单 / 专家模式
- M3 Excel 导入导出 / UI 对齐原型
- M4 自助改密 / 建库结构模板 / LAM 并存回归
