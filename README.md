# NewLDAP

> 面向中文用户的 OpenLDAP 网页管理界面 —— 让不懂 LDAP 的人也能管好 OpenLDAP。

## 这是什么 / 这不是什么

**NewLDAP 不是一个新的 LDAP 服务器，也不提供任何新的目录服务功能。** 它本质上只是 **OpenLDAP 的一个 Web 界面**：你的数据仍然全部存在 OpenLDAP 里，由 OpenLDAP 提供认证、ACL、复制等全部目录能力；NewLDAP 只是通过标准 LDAP 协议读写目录，给你一个更友好的中文操作界面。

- ❌ 不是目录服务器 —— 目录服务仍由 OpenLDAP 承担
- ❌ 不持有数据 —— 没有影子数据库，目录本身就是唯一数据源，界面上看到的就是目录里的真实内容
- ✅ 是一个管理台 —— 部门 / 人员 / 用户组 / 条目编辑 / Excel 导入导出 / 自助改密 / 审计日志
- ✅ 可以与其他标准 LDAP 客户端（LAM、phpLDAPadmin、堡垒机等）**安全并存**，共管同一目录

**为什么做它**：OpenLDAP 功能强大但概念门槛高（DN、objectClass、schema、LDIF……），现有管理工具（如 LAM、phpLDAPadmin）界面偏老、以英文为主，go-ldap-admin 又会把用户平铺进自己的影子库、丢失组织架构。NewLDAP 面向国内运维场景设计：中文优先、保留任意嵌套的组织架构、向导式操作，把 LDAP 概念藏在合理的默认行为后面。

## 功能总览

| 模块 | 能力 |
| --- | --- |
| 登录与会话 | 登录即真实 LDAP bind，权限 = 服务端 ACL，工具不额外设限；凭据 AES-GCM 加密落盘 |
| 目录树 | 全 RDN 树、部门拖拽排序、OU 中文名、行内新建人员 |
| 组织与人员 | uid 全组织唯一 + 拼音自动生成 + 防抖查重、批量调部门、禁用/启用（服务端判定） |
| 用户组 | 权限组（groupOfNames）/ 登录组（posixGroup 自动分配 gidNumber）、成员搜索添加 |
| 条目编辑 | 简单/专家双模式：schema 表单、对象类可视化编辑、LDIF 视图、entryCSN 冲突拦截 |
| Excel 导入导出 | 中文模板 → 计划预览（行级错误定位）→ 执行 → 初始密码清单；导出脱敏 |
| 帮助中心 | 新手引导、LDAP 教学（含可执行的过滤器实验器）、堡垒机/零信任/VPN/SSO 对接指引（一键生成只读对接账号 + 凭据自测 + ACL 样例）、术语词典 |
| 自助改密 | `/password` 独立页面，员工自行改密，ppolicy 错误白话翻译，双维度限流 |
| 审计日志 | 全操作留痕，JSONL 落盘，可导出 CSV |
| 演示模式 | `--mock` 内置示例目录，零依赖体验全部功能 |

## 快速开始

### 方式 A：Docker 一键体验（OpenLDAP + NewLDAP + phpLDAPadmin 三件套）

```bash
cd deploy && docker compose up -d
# NewLDAP 管理台  http://localhost:8080
# phpLDAPadmin   http://localhost:8081（交叉验证"目录即真相"）
# 自助改密       http://localhost:8080/password
```

### 方式 B：接入你已有的 OpenLDAP（推荐的生产用法）

不需要重复部署目录服务。用根目录 `Dockerfile` 构建镜像（或交叉编译二进制后用 `deploy/Dockerfile.binary`），通过环境变量指向现有目录：

```yaml
environment:
  LDAP_URL: ldap://<你的openldap>:389
  LDAP_BIND_DN: cn=admin,dc=example,dc=com
  LDAP_BIND_PASSWORD: <管理员密码>
  LDAP_BASE_DN: dc=example,dc=com
  NEWLDAP_SECRET_KEY: <64位hex或任意强随机串>   # 凭据 AES-256-GCM 加密，生产必配
```

完整生产编排见 [deploy/docker-compose.prod.yml](deploy/docker-compose.prod.yml) 与 [deploy/README-deploy.md](deploy/README-deploy.md)。

### 方式 C：源码运行（开发 / 演示）

```bash
# 演示模式：内置示例目录，无需任何外部依赖（Go 1.23+）
go run ./cmd/server --mock --addr :8080
# 账号：cn=admin,dc=example,dc=cn / admin123（员工 zhangwei 等 / Passw0rd!）

# 前端开发（Node 20+）
cd web && npm install && npm run dev
```

## 文档

| 文档 | 内容 |
| --- | --- |
| [使用手册](docs/11-使用手册.md) | 面向管理员的日常使用说明（部门/人员/组/导入导出/对接指引） |
| [部署运维手册](docs/10-部署运维手册.md) | 生产部署、schema 注入、升级流程、备份与排错 |
| [部署包说明](deploy/README-deploy.md) | 三件套编排、接入已有 OpenLDAP、踩坑记录 |

## 安全说明

- 对接外部系统请使用帮助中心创建的**专用只读账号**，配合样例 ACL 限制为只读；不要把 admin 凭据填入第三方系统。
- `userPassword` 只在服务端判定 / 哈希存储，任何接口不回显。
- 仓库内 compose 与种子数据中的口令均为**演示值**，生产环境请全部更换并通过环境变量注入。

## 许可

暂未设定（待项目所有者确定）。
