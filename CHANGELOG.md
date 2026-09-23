# Changelog

本项目的显著变更记录。版本号在首次正式发布前为开发迭代线。

## [Unreleased] / v1.x 开发线（截至 2026-09-23）

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
- 各阶段报告见 docs/05–09。
