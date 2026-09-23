// M5：LDAP 服务端错误的集中翻译（R11.4 权限降级 + 常见错误友好化）。
package api

import (
	"errors"
	"net/http"
	"strings"

	ldap "github.com/go-ldap/ldap/v3"
)

// writeLdapErr 把目录服务端返回的错误翻译为中文白话与合适的 HTTP 状态。
// 已处理返回 true；非 LDAP 错误返回 false（调用方自行处理）。
func writeLdapErr(w http.ResponseWriter, err error, action string) bool {
	var le *ldap.Error
	if !errors.As(err, &le) {
		// 不是结构化 LDAP 错误：按字符串兜底识别（ldapclient 包装保留了原文）
		msg := err.Error()
		switch {
		case strings.Contains(msg, "存在") && strings.Contains(msg, "已"):
			writeErr(w, http.StatusConflict, action+"失败：同名条目已存在")
			return true
		case strings.Contains(msg, "不存在"):
			writeErr(w, http.StatusNotFound, action+"失败：目标条目不存在（可能刚被他人删除），请刷新后重试")
			return true
		}
		return false
	}
	switch le.ResultCode {
	case ldap.LDAPResultInsufficientAccessRights,
		8,  // strongerAuthRequired
		13: // confidentialityRequired（要求 TLS）
		// R11.4：无权操作的优雅降级——明确告诉用户原因与出路，而不是错误码
		writeErr(w, http.StatusForbidden,
			"当前账号无权"+action+"（服务端 ACL 拒绝）。如确需操作，请联系目录管理员授予相应权限。")
	case ldap.LDAPResultNoSuchObject:
		writeErr(w, http.StatusNotFound, action+"失败：目标条目不存在（可能刚被他人删除），请刷新后重试")
	case ldap.LDAPResultEntryAlreadyExists:
		writeErr(w, http.StatusConflict, action+"失败：同名条目已存在，请换一个名称")
	case ldap.LDAPResultNotAllowedOnNonLeaf:
		writeErr(w, http.StatusConflict, action+"失败：该条目下还有子条目，需先处理子条目")
	case ldap.LDAPResultInvalidCredentials:
		writeErr(w, http.StatusUnauthorized, action+"失败：账号或密码错误")
	case ldap.LDAPResultObjectClassViolation, ldap.LDAPResultConstraintViolation:
		writeErr(w, http.StatusBadRequest, action+"失败：目录结构校验未通过（缺少必填属性或违反 schema 约束）："+le.Err.Error())
	default:
		writeErr(w, http.StatusBadGateway, action+"失败："+le.Err.Error())
	}
	return true
}
