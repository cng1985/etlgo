package protocol

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeOK           = 200
	CodeBadRequest   = 400
	CodeUnauthorized = 401
	CodeForbidden    = 403
	CodeNotFound     = 404
	CodeConflict     = 409
	CodeLimited      = 429
	CodeError        = 500
)

type Response[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type PageBody struct {
	Code         int    `json:"code"`
	Msg          string `json:"msg"`
	Data         any    `json:"data"`
	Page         int    `json:"page"`
	Size         int    `json:"size"`
	Total        int64  `json:"total"`
	TotalPage    int    `json:"totalPage"`
	ResponseType string `json:"responseType"`
}

type Error struct {
	Code int
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

func Bad(msg string) error          { return &Error{Code: CodeBadRequest, Msg: msg} }
func Unauthorized(msg string) error { return &Error{Code: CodeUnauthorized, Msg: msg} }
func Forbidden(msg string) error    { return &Error{Code: CodeForbidden, Msg: msg} }
func NotFound(msg string) error     { return &Error{Code: CodeNotFound, Msg: msg} }

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response[any]{Code: CodeOK, Msg: "success", Data: data})
}

func Page(c *gin.Context, data any, page, size int, total int64) {
	if data == nil {
		data = []any{}
	}
	totalPage := 0
	if size > 0 && total > 0 {
		totalPage = int((total + int64(size) - 1) / int64(size))
	}
	c.JSON(http.StatusOK, PageBody{
		Code:         CodeOK,
		Msg:          "success",
		Data:         data,
		Page:         page,
		Size:         size,
		Total:        total,
		TotalPage:    totalPage,
		ResponseType: "page",
	})
}

func Fail(c *gin.Context, code int, msg string) {
	if code < 400 {
		code = CodeError
	}
	c.JSON(code, Response[any]{Code: code, Msg: msg, Data: nil})
}

func NormalizePage(page, size int) (int, int, error) {
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}
	if page < 1 {
		return 0, 0, Bad("page 必须大于等于 1")
	}
	if size < 1 || size > 200 {
		return 0, 0, Bad("size 必须在 1 到 200 之间")
	}
	return page, size, nil
}
