# 生产形态部署（compose 三件套）

部署包在目标机 `~/newldap-deploy/`（示例主机 192.168.233.132）。

## 文件
- `docker-compose.prod.yml`：openldap + newldap + phpldapadmin（newldap 用二进制构建，前端已内嵌）
- `Dockerfile.binary`：最小镜像（alpine + 静态二进制；本地交叉编译 `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o newldap-server ./cmd/server`）
- `schema-newldap.ldif`：ouOrder 自定义属性 + newldapOUExtras 辅助类（树的部门排序持久化需要；先于 seed 注入：`ldapadd -Y EXTERNAL -H ldapi:// -f schema.ldif`，容器内执行）
- `seed.ldif`：测试数据（5 部门含中文名/排序、15 人员 密码 Passw0rd!、3 组）
- `run-tests.sh`：29 项功能冒烟（登录/树/搜索/建人/uid 查重/改名/组/自助改密/entryCSN 409/密码不泄漏/禁用启用/phpLDAPadmin）。注意：加成员请求须用嵌套 `changes:{add,remove}` 结构（旧扁平结构会报"没有成员变更"）；基线为 15 人，如有历史测试残留需先清理。

## 2026-09-23 更新（三）：PLDA 同源反代 + Base DN 网页管理
- NewLDAP 以 /plda/ 前缀反向代理 phpLDAPadmin（桥接页自动代填登录，Cookie 同源）；外层路由已注册，升级二进制即生效。
- 系统设置 → 目录连接 → 修改 Base DN：保存写 /data/config.yaml 并置 wizard_saved: true，**重启后优先于 compose 的 LDAP_* 环境变量**（source 由 env 翻转为 file）。

## 2026-09-23 更新（二）：系统设置 + CSRF/换绑修复
- newldap 服务挂载 newldap-data:/data 卷：系统设置策略（data/config.yaml）与审计日志跨容器重建持久化。
- 仓库 docker-compose.prod.yml 以服务器实际运行的 docker-compose.yml 为准回填（此前仓库版本的健康检查还是坏的 1389 旧版，勿再用旧版覆盖服务器）。
- 升级流程不变：scp 二进制 → docker compose build newldap && up -d newldap；跑 run-tests.sh 时注意：自助改密用例连续失败会触发账号级限流（约 10 分钟窗口），失败后等待窗口再重跑。

## 2026-09-23 更新（一）：帮助中心（新手引导 / LDAP 教学 / 对接指引 / 术语词典）
- 交叉编译上传 `newldap-server` → `docker compose build newldap && docker compose up -d newldap` 即可热更新（前端已内嵌）。
- 新增后端端点：`POST /api/v1/integration/bindacct`（对接只读账号：位置可选，默认 `ou=services` 自动创建；organizationalRole + simpleSecurityObject 标准类，真实 OpenLDAP 无需注入 schema）、`POST /api/v1/integration/bindtest`（凭据自测，错误密码返回 49 白话）。
- 29 项冒烟 + 帮助中心生产实测全通过（实验器对生产目录真实查询命中 15 人；对接地址按浏览器地址自动推断为 ldap://<host>:389）。

## 踩坑记录（务必保留）
1. osixia/openldap 1.5.0 内部端口是 **389**（不是文档示例里的 1389）。
2. 默认 ACL 禁匿名查询：**健康检查必须带 -D admin -w 密码**，否则 container 永远 unhealthy。
3. 改过环境变量后必须 `docker compose down -v` 清卷重来（首启引导不会重跑）。
4. 真实 OpenLDAP 会拒绝未定义属性（ouOrder）：先注入 schema-newldap.ldif，OU 条目加 `objectClass: newldapOUExtras`。
5. 宿主机端口冲突：8080 被邮件服务（stalwart）占用，NewLDAP 暴露在 **18080**，phpLDAPadmin 8081。

## 访问
- NewLDAP 管理台：http://192.168.233.132:18080 （admin cn=admin,dc=example,dc=cn / admin123）
- phpLDAPadmin：http://192.168.233.132:8081 （同一管理员）
- 员工自助改密：http://192.168.233.132:18080/password
- 侧栏 phpLDAPadmin 按钮指向 8081（NEWLDAP_PLDA_URL）
