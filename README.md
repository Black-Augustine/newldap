# NewLDAP

> 面向中文用户的 OpenLDAP 网页管理台 —— 目录即真相，无影子数据库，与 LAM / phpLDAPadmin 安全并存。

## 项目定位

一个中文风格的 OpenLDAP Web 管理界面：比 LAM 更简单易用、符合国内用户的操作习惯，同时**完整保留 OpenLDAP 的目录架构模式**（任意嵌套 OU、自由 DIT、不做影子数据库），让完全不懂 LDAP 的人也能快速上手。

与 go-ldap-admin 的本质区别：不建影子数据库、不把用户平铺在 `ou=people` 下，**目录本身就是唯一数据源**，本工具只是目录的一个更友好的读写视图。

## 核心理念

1. **目录即真相（Directory as the Source of Truth）**——工具不持有任何业务数据库，所有数据实时读写 LDAP，目录里是什么，界面上就是什么。
2. **双模式界面**——简单模式用业务语言（部门 / 人员 / 用户组），专家模式直接暴露 DN / objectClass / 原始属性 / LDIF，新手和专家各取所需。
3. **新手友好**——连接向导、建库向导、结构模板、白话术语解释、帮助中心（新手引导 / LDAP 教学 / 对接指引），把 LDAP 概念藏在合理的默认行为后面。
4. **目录好公民**——只通过标准 LDAP 操作修改目录（属性级修改），与 LAM 等其他标准客户端可安全并存、共管同一目录（entryCSN 乐观并发 + 自动化回归测试保障）。

## 功能总览

| 模块 | 能力 |
| --- | --- |
| 登录与会话 | 登录即真实 bind，权限 = 服务端 ACL，工具不额外设限；凭据 AES-GCM 加密落盘 |
| 目录树 | phpLDAPadmin 风格全 RDN 树、根节点查看全员、部门拖拽排序（ouOrder）、OU 中文名（第二个 ou 值）、行内新建人员 |
| 组织与人员 | uid 全组织唯一 + 拼音自动生成 + 防抖查重、批量移动部门、自定义列与列排序、列宽拖动、禁用态真实化（密码服务端判定，不出网） |
| 用户组 | 场景化建组（权限组 groupOfNames / 登录组 posixGroup 自动分配 gidNumber）、成员远程搜索添加、人数上限与限高 |
| 条目编辑 | 双模式抽屉：必填/可选/操作属性分页、对象类可视化编辑（主类/辅助类）、LDIF 视图、entryCSN 冲突拦截（409） |
| Excel 导入导出 | 中文模板 → 上传 → 计划预览（行级错误定位）→ 执行 → 初始密码清单；导出脱敏 |
| 帮助中心 | 新手引导（6 步实时清单 + 聚光灯导览）、LDAP 教学（9 课真实数据举例 + 可执行查询的过滤器实验器）、**对接指引**（堡垒机/零信任/VPN/SSO 信息卡一键复制、对接只读账号一键创建 + 凭据自测 + ACL 样例）、术语词典 |
| 自助改密 | `/password` 独立轻量页面，员工自 bind 验证后改密，ppolicy 错误白话翻译，双维度限流 |
| 审计日志 | 全操作留痕（含 LAM 侧可感知的写操作），JSONL 落盘，可导出 |
| 演示模式 | `--mock` 内置进程内 LDAP 替身（BER 线协议真实收发），零依赖演示 / 开发 / e2e |

## 产品形态

| 形态 | 说明 |
| --- | --- |
| 伴随部署 | 与 OpenLDAP 通过 docker compose 一键起栈，环境变量预配置，开箱即"类原生"体验 |
| 独立运维工具 | 独立部署，通过连接向导 + bind 账号对接任意第三方 LDAP |
| 自助改密入口 | `/password` 独立页面，员工自助改密，无需管理员参与 |

## 快速开始（开发）

```bash
# 后端（Go 1.23）——演示模式：内置示例目录，无需外部依赖
go run ./cmd/server --mock --addr :8080
#   演示环境变量：NEWLDAP_MOCK_ADDR=127.0.0.1:3890（固定 mock 端口）
#               NEWLDAP_PLDA_URL=http://localhost:8081（侧栏 phpLDAPadmin 跳转）
# 账号：cn=admin,dc=example,dc=cn / admin123（员工 zhangwei 等 / Passw0rd!）

# 独立部署（无 --mock）：首次访问 Web 自动进入连接向导
#   服务器地址 → rootDSE 自动探测 Base DN → 验证管理账号 → 保存（密码 AES-GCM 加密）
#   建议设置 NEWLDAP_SECRET_KEY 环境变量（未设置则明文保存并警告）
go run ./cmd/server --addr :8080

# 前端开发模式（热更新，代理到 8080）
cd web && npm install && npm run dev      # http://localhost:5173
npm run typecheck                         # vue-tsc

# 测试
go test ./...                             # 单元 + 线协议替身 + e2e
LDAP_IT_URL=ldap://… go test ./internal/ldapclient/ -run Integration -v   # 真实 OpenLDAP（CI 自动跑）

# Docker（伴随形态样例）
cd deploy && docker compose up -d
```

生产部署（含真实 OpenLDAP 注意事项、schema 注入、29 项冒烟、升级流程）见 [docs/10-部署运维手册.md](docs/10-部署运维手册.md)。

### Docker 一键体验（OpenLDAP + NewLDAP + phpLDAPadmin）

仓库 `deploy/` 目录内置三件套的伴随部署编排，一条命令拉起完整环境（含演示数据）：

```bash
cd deploy && docker compose up -d
# NewLDAP 管理台  http://localhost:8080   （连接向导自动完成，或环境变量预配置）
# phpLDAPadmin   http://localhost:8081   （交叉验证"目录即真相"）
# 自助改密       http://localhost:8080/password
```

接入**已有** OpenLDAP 无需重复部署目录服务：删除 compose 中的 openldap 服务，用 `LDAP_URL / LDAP_BIND_DN / LDAP_BIND_PASSWORD / LDAP_BASE_DN` 四个环境变量指向现有目录即可，详见 [deploy/README-deploy.md](deploy/README-deploy.md)。

## 仓库结构

```
cmd/server/          # 服务入口（HTTP + 内嵌前端 + --mock 演示模式）
internal/
  api/               # REST 路由（m1 基础/m2 人员组/m3 导入导出审计/m4 自助改密/m6 对接指引）
  auth/              # 会话（登录即 bind，会话连接即权限；滑动过期）
  audit/             # JSONL 审计日志（login/setup/create/modify/move/delete/bindacct…）
  config/            # YAML + 环境变量 + AES-GCM 凭据加密（NEWLDAP_SECRET_KEY）
  directory/         # 树浏览计数、条目读写、entryCSN 冲突检测（409）、modrdn、删除保护/级联
  people/            # 人员领域层（uid 唯一、拼音生成、复姓推导、禁用态判定）
  group/             # 用户组领域层（权限组/登录组、成员属性级维护）
  importer/          # Excel 导入执行（多级部门路径、计划 diff）
  selfservice/       # 员工自助改密（自 bind 验证 + ppolicy 白话 + 限流）
  excel/             # 中文模板生成/解析（行级错误定位）
  ldapclient/        # go-ldap 封装：TLS/分页(宽容)/modrdn/rootDSE/subschema
  ldaptest/          # 进程内 LDAP 测试替身（BER 线协议，RFC4511 过滤器求值）
  planexec/          # 计划引擎 diff 内核（Excel 导入 / v2 同步共用）
  schema/            # RFC4512 subschema 解析器
web/                 # Vue 3 + Element Plus 按需 + lucide 线性图标（构建产物内嵌进 Go 二进制）
deploy/              # compose 伴随样例 + 生产部署（二进制镜像/schema/种子数据/29 项冒烟）
docs/                # 项目文档（需求 → 架构 → 验证报告 → 部署/使用手册）
prototype/           # 高保真 HTML 原型（含帮助中心原型，离线可开）
test/e2e/            # HTTP 层端到端冒烟（真实替身目录）
.github/workflows/   # CI：vet + 全量测试 / 真实 OpenLDAP 集成 / 前端 typecheck+build / 镜像
```

## 文档索引

| 文档 | 内容 |
| --- | --- |
| [需求清单](docs/01-需求清单.md) | 全量功能 / 非功能需求、优先级与版本归属 |
| [MVP 范围定义](docs/02-MVP范围定义.md) | MVP 边界、场景故事、验收标准、交付物 |
| [开发计划](docs/03-开发计划.md) | 里程碑排期、后续版本路线、风险与对策 |
| [技术架构设计](docs/04-技术架构设计.md) | 技术栈、模块划分、关键设计决策、M0 验证结论 |
| [M0–M4 验证报告](docs/05-M0验证报告.md) | 各里程碑技术验证结论与证据（05–09 五份） |
| [全面检查报告](docs/review-2026-09-22-全面检查报告.md) | 组织团队全面检查：11 项修复 + 待办清单 |
| [帮助中心规划](docs/plan-2026-09-23-新手引导-LDAP教学-对接指引.md) | 新手引导 / LDAP 教学 / 对接指引 的产品规划 |
| [部署运维手册](docs/10-部署运维手册.md) | 生产部署、schema 注入、升级流程、备份与排错 |
| [使用手册](docs/11-使用手册.md) | 面向管理员的日常使用说明（含对接指引用法） |
| [产品原型](prototype/README.md) | HTML 高保真交互原型（单页、离线可用、单色线性图标） |
| [CHANGELOG](CHANGELOG.md) | 版本变更记录 |

## 与同类产品的差异

| 维度 | LAM | go-ldap-admin | 本项目 |
| --- | --- | --- | --- |
| 数据存储 | 直连 LDAP | 自建 MySQL 影子库同步 | 直连 LDAP，无影子库 |
| 目录结构 | 自由 DIT | 用户平铺 `ou=people` | 自由 DIT，任意嵌套 OU |
| 界面语言 / 风格 | 英文为主，界面偏老 | 中文，现代化 | 中文优先，符合国人操作习惯 |
| 新手可用性 | 需懂 LDAP 概念 | 一般 | 向导 + 双模式 + 帮助中心（教学/对接指引），零基础可用 |
| 与其他客户端并存 | — | 影子库易失同步 | 明确设计目标，entryCSN 冲突拦截 + 自动化回归测试 |
| 批量导入 | CSV 为主 | 有 | Excel（中文模板），后续飞书/企微/钉钉 |

## 安全说明

- 对接外部系统请使用帮助中心创建的**专用只读账号**（organizationalRole + simpleSecurityObject），配合样例 ACL 限制为只读；不要把 admin 凭据填入第三方系统。
- `userPassword` 在服务端判定 / 哈希存储，任何接口不回显（有 e2e 防泄漏回归测试）。
- 仓库内 compose / 种子数据中的口令均为**演示值**，生产环境请全部更换并通过 `.env` 或环境变量注入。

## 许可

暂未设定（待项目所有者确定）。
