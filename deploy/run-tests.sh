#!/bin/bash
# NewLDAP 真实环境功能测试（在目标机上运行）
B=http://localhost:18080
CK=/tmp/nl-cookie.txt
PASS=0; FAIL=0
ok()   { PASS=$((PASS+1)); echo "  ✓ $1"; }
bad()  { FAIL=$((FAIL+1)); echo "  ✗ $1  -->  $2"; }
check(){ if [ "$1" = "$2" ]; then ok "$3"; else bad "$3" "want=$2 got=$1"; fi }

echo "== 1. 登录与基础 =="
curl -s -c $CK -X POST $B/api/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"bindDN":"cn=admin,dc=example,dc=cn","password":"admin123"}' | grep -q bindDN && ok "管理员登录" || bad "管理员登录" "no response"
R=$(curl -s -b $CK "$B/healthz"); echo "$R" | grep -q '"ldap":"up"' && ok "healthz ldap=up（非 mock）" || bad "healthz" "$R"

echo "== 2. 树 / 列表 / 中文搜索 =="
R=$(curl -s -b $CK "$B/api/v1/tree?containers=1")
echo "$R" | grep -q '"displayName":"技术部"' && ok "树含部门中文名（技术部）" || bad "树中文名" "$R"
echo "$R" | grep -q '"ou=hr"' && echo "$R" | python3 -c "
import json,sys; d=json.load(sys.stdin)
ns=[c['rdn'] for c in d['children']]
assert ns.index('ou=tech') < ns.index('ou=market') < ns.index('ou=hr'), ns
print('ok')" >/dev/null 2>&1 && ok "树按 ouOrder 排序（tech→market→hr）" || bad "树排序" "$R"
R=$(curl -s -b $CK "$B/api/v1/people/list?base=dc=example,dc=dc=cn" | head -c 0; curl -s -b $CK "$B/api/v1/people/list")
echo "$R" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d['total'])" > /tmp/nl_total 2>/dev/null
check "$(cat /tmp/nl_total 2>/dev/null)" "15" "人员列表 15 人（含禁用态接口）"
R=$(curl -s -b $CK "$B/api/v1/search?q=%E5%BC%A0%E4%BC%9F&attr=dn")
echo "$R" | grep -q 'uid=zhangwei' && ok "中文搜索「张伟」命中" || bad "中文搜索" "$R"

echo "== 3. 建人 / uid 查重 / 改名 =="
R=$(curl -s -b $CK -X POST $B/api/v1/people -H 'Content-Type: application/json' \
  -d '{"cn":"\u8d75\u654f","uid":"zhaomin","ou":"ou=tech,dc=example,dc=cn","mail":"","mobile":"","employeeNumber":"E9999","title":"","initialPassword":""}')
echo "$R" | grep -q 'uid=zhaomin' && ok "创建赵敏 uid=zhaomin（sn 自动取姓）" || bad "建人" "$R"
R=$(curl -s -b $CK -X POST $B/api/v1/people -H 'Content-Type: application/json' \
  -d '{"cn":"\u91cd\u590d","uid":"zhaomin","ou":"ou=product,dc=example,dc=cn","mail":"","mobile":"","employeeNumber":"","title":"","initialPassword":""}')
echo "$R" | grep -q '\u5168\u7ec4\u7ec7\u552f\u4e00\|已被使用' && ok "uid 全组织唯一拦截（跨部门重复）" || bad "uid 查重" "$R"
R=$(curl -s -b $CK "$B/api/v1/entry?dn=uid%3Dzhaomin%2Cou%3Dtech%2Cdc%3Dexample%2Cdc%3Dcn")
echo "$R" | python3 -c "import json,sys; print(json.load(sys.stdin)['attrs']['sn'][0])" > /tmp/nl_sn 2>/dev/null
check "$(cat /tmp/nl_sn 2>/dev/null)" "赵" "姓氏 sn=赵（复姓规则）"
R=$(curl -s -b $CK -X POST $B/api/v1/entry/move -H 'Content-Type: application/json' \
  -d '{"dn":"uid=zhaomin,ou=tech,dc=example,dc=cn","newRDN":"uid=zhaomin2"}')
echo "$R" | grep -q 'uid=zhaomin2' && ok "改账号 zhaomin→zhaomin2（DN 重命名）" || bad "改名" "$R"

echo "== 4. 组操作 =="
R=$(curl -s -b $CK -X POST $B/api/v1/groups -H 'Content-Type: application/json' \
  -d '{"name":"test-group","scenario":"\u6743\u9650\u7ec4","members":["uid=lina,ou=tech,dc=example,dc=cn"],"description":"\u6d4b\u8bd5\u7ec4"}')
echo "$R" | grep -q 'cn=test-group' && ok "创建权限组 test-group" || bad "建组" "$R"
R=$(curl -s -b $CK -X POST $B/api/v1/groups/members -H 'Content-Type: application/json' \
  -d '{"dn":"cn=test-group,ou=groups,dc=example,dc=cn","changes":{"add":["uid=chenjing,ou=tech,dc=example,dc=cn"],"remove":[]}}')
[ -z "$R" ] || echo "$R" | grep -q error && bad "加成员" "$R" || ok "组加成员 chenjing"
R=$(curl -s -b $CK "$B/api/v1/groups" | python3 -c "import json,sys; gs=json.load(sys.stdin)['groups']; print([g for g in gs if g['name']=='vpn-access'][0]['memberCount'])" 2>/dev/null)
check "$R" "3" "vpn-access 组成员数 3（组列表反查）"

echo "== 5. 自助改密 =="
R=$(curl -s -X POST $B/api/v1/selfservice/password -H 'Content-Type: application/json' \
  -d '{"account":"zhangwei","oldPassword":"Passw0rd!","newPassword":"NewP@ss123"}')
echo "$R" | grep -q '"ok"' && ok "张伟自助改密 Passw0rd!→NewP@ss123" || bad "自助改密" "$R"
docker exec newldap-deploy-openldap-1 ldapwhoami -x -H ldap://localhost:389 \
  -D uid=zhangwei,ou=tech,dc=example,dc=cn -w 'NewP@ss123' 2>/dev/null | grep -q zhangwei && ok "新密码 bind 验证通过（LDAP 侧）" || bad "新密码bind" "ldapwhoami failed"
curl -s -X POST $B/api/v1/selfservice/password -H 'Content-Type: application/json' \
  -d '{"account":"zhangwei","oldPassword":"Passw0rd!","newPassword":"whatever1"}' | grep -q error && ok "旧密码已失效（防重放）" || bad "旧密码" "should fail"
curl -s -X POST $B/api/v1/selfservice/password -H 'Content-Type: application/json' \
  -d '{"account":"zhangwei","oldPassword":"NewP@ss123","newPassword":"Passw0rd!"}' | grep -q '"ok"' && ok "改回原密码（还原测试数据）" || bad "改回" "fail"

echo "== 6. 并发冲突（entryCSN） =="
CSN=$(curl -s -b $CK "$B/api/v1/entry?dn=uid%3Dlina%2Cou%3Dtech%2Cdc%3Dexample%2Cdc%3Dcn" | python3 -c "import json,sys; print(json.load(sys.stdin)['entryCSN'])")
curl -s -b $CK -X PUT $B/api/v1/entry -H 'Content-Type: application/json' \
  -d "{\"dn\":\"uid=lina,ou=tech,dc=example,dc=cn\",\"expectedCSN\":\"$CSN\",\"changes\":[{\"op\":\"replace\",\"attr\":\"title\",\"vals\":[\"资深工程师\"]}]}" | grep -q entryCSN && ok "正常保存（title=资深工程师）" || bad "正常保存" "fail"
curl -s -o /dev/null -w '%{http_code}' -b $CK -X PUT $B/api/v1/entry -H 'Content-Type: application/json' \
  -d "{\"dn\":\"uid=lina,ou=tech,dc=example,dc=cn\",\"expectedCSN\":\"$CSN\",\"changes\":[{\"op\":\"replace\",\"attr\":\"title\",\"vals\":[\"X\"]}]}" > /tmp/nl_409
check "$(cat /tmp/nl_409)" "409" "过期 CSN 保存被 409 拦截（LAM 并存保护）"

echo "== 7. 安全：密码不泄漏 =="
R=$(curl -s -b $CK "$B/api/v1/entry?dn=uid%3Dzhangwei%2Cou%3Dtech%2Cdc%3Dexample%2Cdc%3Dcn")
echo "$R" | grep -qi userPassword && bad "详情泄漏 userPassword" "$R" || ok "详情不泄漏 userPassword"
R=$(curl -s -b $CK "$B/api/v1/search?q=zhangwei&attr=userPassword&attr=cn")
echo "$R" | grep -qi userPassword && bad "搜索泄漏 userPassword" "$R" || ok "搜索不泄漏 userPassword"

echo "== 8. 禁用/启用 =="
curl -s -b $CK -X POST $B/api/v1/people/disable -H 'Content-Type: application/json' \
  -d '{"dn":"uid=wangqiang,ou=tech,dc=example,dc=cn"}' | grep -q '"ok"' && ok "禁用王强" || bad "禁用" "fail"
R=$(curl -s -b $CK "$B/api/v1/people/list" | python3 -c "import json,sys; ps=json.load(sys.stdin)['people']; print([p['disabled'] for p in ps if 'wangqiang' in p['dn']][0])" 2>/dev/null)
check "$R" "True" "列表禁用态=True"
docker exec newldap-deploy-openldap-1 ldapwhoami -x -H ldap://localhost:389 \
  -D uid=wangqiang,ou=tech,dc=example,dc=cn -w 'Passw0rd!' >/dev/null 2>&1 && bad "禁用后仍可登录" "should fail" || ok "禁用后登录被拒（LDAP 侧验证）"
curl -s -b $CK -X POST $B/api/v1/people/enable -H 'Content-Type: application/json' \
  -d '{"dn":"uid=wangqiang,ou=tech,dc=example,dc=cn"}' | grep -q '"ok"' && ok "启用王强（还原）" || bad "启用" "fail"

echo "== 9. phpLDAPadmin =="
C=$(curl -s -o /dev/null -w '%{http_code}' http://localhost:8081/)
check "$C" "200" "phpLDAPadmin 页面 HTTP 200"
curl -s http://localhost:8081/ | grep -qi 'phpLDAPadmin\|login' && ok "phpLDAPadmin 登录页渲染" || bad "PLDA 页面" "no content"

echo "== 10. 清理测试数据 =="
curl -s -b $CK -X POST $B/api/v1/entry/delete -H 'Content-Type: application/json' \
  -d '{"dn":"uid=zhaomin2,ou=tech,dc=example,dc=cn","cascade":false}' | grep -q '"ok"' && ok "删除测试人员 zhaomin2" || bad "删测试人员" "fail"
curl -s -b $CK -X POST $B/api/v1/entry/delete -H 'Content-Type: application/json' \
  -d '{"dn":"cn=test-group,ou=groups,dc=example,dc=cn","cascade":false}' | grep -q '"ok"' && ok "删除测试组 test-group" || bad "删测试组" "fail"
# 还原 lina 职务
curl -s -b $CK -X PUT $B/api/v1/entry -H 'Content-Type: application/json' \
  -d '{"dn":"uid=lina,ou=tech,dc=example,dc=cn","changes":[{"op":"replace","attr":"title","vals":["\u5e94\u7528\u7814\u53d1\u5de5\u7a0b\u5e08"]}]}' >/dev/null

echo ""
echo "=========================================="
echo "  结果：通过 $PASS 项 / 失败 $FAIL 项"
echo "=========================================="
