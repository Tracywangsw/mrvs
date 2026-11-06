package apperr

import (
	"fmt"
	"net/http"
)

// 错误码，与设计文档 7.3 保持一致。
const (
	CodeOK                   = "OK"
	CodeValidation           = "VALIDATION_ERROR"
	CodeNotFound             = "NOT_FOUND"
	CodeInvalidState         = "INVALID_STATE"
	CodeDuplicateAction      = "DUPLICATE_ACTION"
	CodeBusinessRule         = "BUSINESS_RULE_VIOLATION"
	CodeHCPHospitalMismatch  = "HCP_HOSPITAL_MISMATCH"
	CodeTimeOrderInvalid     = "TIME_ORDER_INVALID"
	CodeTimeOverrideDisabled = "TIME_OVERRIDE_DISABLED"
	CodeMaterialInconsistent = "MATERIAL_INCONSISTENT"
	CodeInternal             = "INTERNAL_ERROR"
)

// Error 表示可直接映射为 HTTP 响应的业务错误。
type Error struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// New 构造业务错误。
func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// Validation 构造参数校验错误（400）。
func Validation(message string) *Error {
	return New(http.StatusBadRequest, CodeValidation, message)
}

// ValidationFields 构造带字段明细的参数校验错误（400）。
func ValidationFields(message string, fields map[string]string) *Error {
	e := Validation(message)
	e.Fields = fields
	return e
}

// NotFound 构造资源不存在错误（404）。
func NotFound(message string) *Error {
	return New(http.StatusNotFound, CodeNotFound, message)
}

// InvalidState 构造状态不允许错误（409）。
func InvalidState(message string) *Error {
	return New(http.StatusConflict, CodeInvalidState, message)
}

// DuplicateAction 构造重复操作错误（409）。
func DuplicateAction(message string) *Error {
	return New(http.StatusConflict, CodeDuplicateAction, message)
}

// BusinessRule 构造业务规则错误（422）。
func BusinessRule(message string) *Error {
	return New(http.StatusUnprocessableEntity, CodeBusinessRule, message)
}

// HCPHospitalMismatch 构造医生与医院不匹配错误（422）。
func HCPHospitalMismatch(message string) *Error {
	return New(http.StatusUnprocessableEntity, CodeHCPHospitalMismatch, message)
}

// TimeOrderInvalid 构造时间倒挂错误（422）。
func TimeOrderInvalid(message string) *Error {
	return New(http.StatusUnprocessableEntity, CodeTimeOrderInvalid, message)
}

// TimeOverrideDisabled 构造演示时间开关未开启错误（422）。
func TimeOverrideDisabled(message string) *Error {
	return New(http.StatusUnprocessableEntity, CodeTimeOverrideDisabled, message)
}

// MaterialInconsistent 构造资料与派发标记不一致错误（422）。
func MaterialInconsistent(message string) *Error {
	return New(http.StatusUnprocessableEntity, CodeMaterialInconsistent, message)
}

// Internal 构造服务端错误（500）。
func Internal(message string) *Error {
	return New(http.StatusInternalServerError, CodeInternal, message)
}
