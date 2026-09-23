# 生产形态部署（OpenLDAP + NewLDAP + phpLDAPadmin 三件套）

本目录提供**伴随形态**的生产部署样例：NewLDAP 与 OpenLDAP、phpLDAPadmin 一键起栈，三个组件共管同一目录，开箱即用。

## 文件说明

| 文件 | 用途 |
| --- | --- |
| `docker-compose.prod.yml` | 生产编排：openldap + newldap + phpldapadmin（newldap 用二进制构建，前端已内嵌进 Go 二进制） |
| `Dockerfile.binary` | 最小运行镜像（alpine + 静态二进制）。需先交叉编译：`CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o deploy/newldap-server ./cmd/server` |
| `schema-newldap.ldif` | ouOrder 自定义属性 + newldapOUExtras 辅助类（目录树的部门排序持久化需要；先于种子数据注入：容器内执行 `ldapadd -Y EXTERNAL -H ldapi:/// -f schema-newldap.ldif`） |
| `seed.ldif` | 演示数据（5 部门含中文名/排序、15 人员 密码 Passw0rd!、3 组），生产环境请勿导入 |
| `run-tests.sh` | 29 项功能冒烟（登录/树/搜索/建人/uid 查重/改名/组/自助改密/entryCSN 409/密码不泄漏/禁用启用/phpLDAPadmin）。注意：加成员请求须用嵌套 `changes:{add,remove}` 结构（旧扁平结构会报"没有成员变更"）；基线为 15 人，如有历史测试残留需先清理 |

> 根目录 `Dockerfile` 为**多阶段全量构建**（前端 + Go，无需预编译二进制），适合 CI 或首次从零构建；
> `Dockerfile.binary` 为**二进制快构**，适合日常迭代热更新（交叉编译 → build → up）。

## 快速部署

```bash
# 1. 交叉编译 Linux 二进制（在有 Go 1.23+ 的开发机执行）
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o deploy/newldap-server ./cmd/server

# 2. 修改 docker-compose.prod.yml 中的密码与 NEWLDAP_PLDA_URL（占位符 <服务器IP>）

# 3. 起栈
cd deploy && docker compose -f docker-compose.prod.yml up -d --build

# 4. 注入自定义 schema（部门排序功能需要，仅需一次）
docker cp schema-newldap.ldif <openldap容器名>:/tmp/
docker exec <openldap容器名> ldapadd -Y EXTERNAL -H ldapi:/// -f /tmp/schema-newldap.ldif
```

## 接入已有 OpenLDAP（不重复部署目录服务）

NewLDAP 是独立运维工具，可对接任意第三方 LDAP：删除 compose 中的 openldap 服务，通过环境变量指向现有目录即可：

```yaml
environment:
  LDAP_URL: ldap://<你的openldap>:389
  LDAP_BIND_DN: cn=admin,dc=example,dc=com
  LDAP_BIND_PASSWORD: <管理员密码>
  LDAP_BASE_DN: dc=example,dc=com
  NEWLDAP_SECRET_KEY: <64位hex或任意强随机串>   # 凭据 AES-256-GCM 加密，生产必配
```

若与现有栈分属不同 compose 项目，把 newldap 加入外部共享网络（`networks: { ldap: { external: true, name: <现有网络名> } }`）即可用容器名直连。

## 踩坑记录（务必保留）

1. osixia/openldap 1.5.0 内部端口是 **389**（不是文档示例里的 1389）。
2. 默认 ACL 禁匿名查询：**健康检查必须带 -D admin -w 密码**，否则容器永远 unhealthy。
3. 改过 openldap 环境变量后必须 `docker compose down -v` 清卷重来（首启引导不会重跑）。
4. 真实 OpenLDAP 会拒绝未定义属性（ouOrder）：先注入 schema-newldap.ldif，OU 条目加 `objectClass: newldapOUExtras`。
5. 生产环境务必设置 `NEWLDAP_SECRET_KEY`，否则连接凭据将明文落盘（启动日志会告警）。
6. 宿主机端口冲突时自行调整映射（样例使用 8080/8081，可按需改为 18080 等）。

## 访问（按实际 IP/端口）

- NewLDAP 管理台：`http://<服务器IP>:8080`（admin：`cn=admin,dc=example,dc=cn`）
- 员工自助改密：`http://<服务器IP>:8080/password`
- phpLDAPadmin：`http://<服务器IP>:8081`（同一管理员，用于交叉验证"目录即真相"）
