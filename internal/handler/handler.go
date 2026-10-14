package handler

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"etlgo/internal/config"
	"etlgo/internal/dto"
	"etlgo/internal/model"
	"etlgo/internal/protocol"
	"etlgo/internal/query"
	"etlgo/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	token     string
	sources   *service.SourceService
	pipelines *service.PipelineService
	datasets  *service.DatasetService
	queries   *query.Executor
}

func New(cfg *config.Config, sources *service.SourceService, pipelines *service.PipelineService, datasets *service.DatasetService, queries *query.Executor) *Handler {
	return &Handler{
		token:     cfg.Token,
		sources:   sources,
		pipelines: pipelines,
		datasets:  datasets,
		queries:   queries,
	}
}

func (h *Handler) Register(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		protocol.OK(c, gin.H{"status": "ok"})
	})
	api := r.Group("/api/v1", h.authenticate, h.rejectTenant)
	api.POST("/health/get", func(c *gin.Context) {
		protocol.OK(c, gin.H{"status": "ok"})
	})
	api.POST("/stats/get", h.stats)

	api.POST("/source/list", h.listSources)
	api.POST("/source/get", h.getSource)
	api.POST("/source/create", h.createSource)
	api.POST("/source/update", h.updateSource)
	api.POST("/source/delete", h.deleteSource)
	api.POST("/source/preview", h.previewSource)

	api.POST("/pipeline/list", h.listPipelines)
	api.POST("/pipeline/get", h.getPipeline)
	api.POST("/pipeline/create", h.createPipeline)
	api.POST("/pipeline/update", h.updatePipeline)
	api.POST("/pipeline/delete", h.deletePipeline)
	api.POST("/pipeline/preview", h.previewPipeline)
	api.POST("/pipeline/execute", h.executePipeline)

	api.POST("/run/list", h.listRuns)
	api.POST("/run/get", h.getRun)

	api.POST("/dataset/list", h.listDatasets)
	api.POST("/dataset/preview", h.previewDataset)
	api.POST("/dataset/export", h.exportDataset)
	api.POST("/dataset/delete", h.deleteDataset)

	api.POST("/file/upload", h.uploadFile)
	api.POST("/query/search", h.querySearch)
	api.POST("/query/list", h.queryList)
	api.POST("/query/one", h.queryOne)
}

type pageRequest struct {
	Page int `json:"page"`
	Size int `json:"size"`
}

type idRequest struct {
	ID uint `json:"id"`
}

type sourceListRequest struct {
	Page int    `json:"page"`
	Size int    `json:"size"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type sourceWriteRequest struct {
	ID     uint            `json:"id"`
	Name   string          `json:"name"`
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

type sourcePreviewRequest struct {
	ID    uint `json:"id"`
	Limit int  `json:"limit"`
}

type pipelineListRequest struct {
	Page    int    `json:"page"`
	Size    int    `json:"size"`
	Name    string `json:"name"`
	Enabled *bool  `json:"enabled"`
}

type pipelineWriteRequest struct {
	ID              uint            `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Enabled         *bool           `json:"enabled"`
	SourceID        uint            `json:"sourceId"`
	Steps           json.RawMessage `json:"steps"`
	TargetType      string          `json:"targetType"`
	TargetConfig    json.RawMessage `json:"targetConfig"`
	ScheduleMinutes int             `json:"scheduleMinutes"`
}

type pipelinePreviewRequest struct {
	SourceID uint            `json:"sourceId"`
	Steps    json.RawMessage `json:"steps"`
	Limit    int             `json:"limit"`
}

type runListRequest struct {
	Page       int    `json:"page"`
	Size       int    `json:"size"`
	Status     string `json:"status"`
	PipelineID uint   `json:"pipelineId"`
}

type datasetListRequest struct {
	Page int    `json:"page"`
	Size int    `json:"size"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type datasetNameRequest struct {
	Name   string `json:"name"`
	Limit  int    `json:"limit"`
	Format string `json:"format"`
}

func (h *Handler) stats(c *gin.Context) {
	if !h.allow(c, query.PermStatsView) {
		return
	}
	stats, err := h.datasets.Stats()
	if err != nil {
		writeError(c, err)
		return
	}
	runs, _ := stats["recentRuns"].([]model.Run)
	stats["recentRuns"] = dto.Runs(runs)
	protocol.OK(c, stats)
}

func (h *Handler) listSources(c *gin.Context) {
	var req sourceListRequest
	if !h.bind(c, &req) {
		return
	}
	h.search(c, query.PermSourceView, "pageForSource", req.Page, req.Size, dropEmpty(map[string]any{
		"name": req.Name,
		"type": req.Type,
	}))
}

func (h *Handler) getSource(c *gin.Context) {
	var req idRequest
	if !h.bind(c, &req) || !h.requireID(c, req.ID) {
		return
	}
	h.one(c, query.PermSourceView, "oneSource", map[string]any{"id": req.ID})
}

func (h *Handler) createSource(c *gin.Context) {
	var req sourceWriteRequest
	if !h.bind(c, &req) || !h.allow(c, query.PermSourceEdit) {
		return
	}
	item, err := h.sources.Create(service.SourceInput{Name: req.Name, Type: req.Type, Config: req.Config})
	if err != nil {
		writeError(c, err)
		return
	}
	protocol.OK(c, dto.Source(*item))
}

func (h *Handler) updateSource(c *gin.Context) {
	var req sourceWriteRequest
	if !h.bind(c, &req) || !h.requireID(c, req.ID) || !h.allow(c, query.PermSourceEdit) {
		return
	}
	item, err := h.sources.Update(req.ID, service.SourceInput{Name: req.Name, Type: req.Type, Config: req.Config})
	if err != nil {
		writeError(c, err)
		return
	}
	protocol.OK(c, dto.Source(*item))
}

func (h *Handler) deleteSource(c *gin.Context) {
	var req idRequest
	if !h.bind(c, &req) || !h.requireID(c, req.ID) || !h.allow(c, query.PermSourceEdit) {
		return
	}
	if err := h.sources.Delete(req.ID); err != nil {
		writeError(c, err)
		return
	}
	protocol.OK(c, nil)
}

func (h *Handler) previewSource(c *gin.Context) {
	var req sourcePreviewRequest
	if !h.bind(c, &req) || !h.requireID(c, req.ID) || !h.allow(c, query.PermSourceView) {
		return
	}
	view, err := h.sources.Preview(c.Request.Context(), req.ID, req.Limit)
	if err != nil {
		writeError(c, err)
		return
	}
	protocol.OK(c, view)
}

func (h *Handler) listPipelines(c *gin.Context) {
	var req pipelineListRequest
	if !h.bind(c, &req) {
		return
	}
	filters := dropEmpty(map[string]any{"name": req.Name})
	if req.Enabled != nil {
		filters["enabled"] = *req.Enabled
	}
	h.search(c, query.PermPipelineView, "pageForPipeline", req.Page, req.Size, filters)
}

func (h *Handler) getPipeline(c *gin.Context) {
	var req idRequest
	if !h.bind(c, &req) || !h.requireID(c, req.ID) {
		return
	}
	h.one(c, query.PermPipelineView, "onePipeline", map[string]any{"id": req.ID})
}

func (h *Handler) createPipeline(c *gin.Context) {
	var req pipelineWriteRequest
	if !h.bind(c, &req) || !h.allow(c, query.PermPipelineEdit) {
		return
	}
	item, err := h.pipelines.Create(toPipelineInput(req))
	if err != nil {
		writeError(c, err)
		return
	}
	protocol.OK(c, dto.Pipeline(*item))
}

func (h *Handler) updatePipeline(c *gin.Context) {
	var req pipelineWriteRequest
	if !h.bind(c, &req) || !h.requireID(c, req.ID) || !h.allow(c, query.PermPipelineEdit) {
		return
	}
	item, err := h.pipelines.Update(req.ID, toPipelineInput(req))
	if err != nil {
		writeError(c, err)
		return
	}
	protocol.OK(c, dto.Pipeline(*item))
}

func (h *Handler) deletePipeline(c *gin.Context) {
	var req idRequest
	if !h.bind(c, &req) || !h.requireID(c, req.ID) || !h.allow(c, query.PermPipelineEdit) {
		return
	}
	if err := h.pipelines.Delete(req.ID); err != nil {
		writeError(c, err)
		return
	}
	protocol.OK(c, nil)
}

func (h *Handler) previewPipeline(c *gin.Context) {
	var req pipelinePreviewRequest
	if !h.bind(c, &req) || !h.allow(c, query.PermPipelineView) {
		return
	}
	view, err := h.pipelines.Preview(c.Request.Context(), service.PreviewInput{
		SourceID: req.SourceID,
		Steps:    req.Steps,
		Limit:    req.Limit,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	protocol.OK(c, view)
}

func (h *Handler) executePipeline(c *gin.Context) {
	var req idRequest
	if !h.bind(c, &req) || !h.requireID(c, req.ID) || !h.allow(c, query.PermExecute) {
		return
	}
	run, err := h.pipelines.StartRun(req.ID, "manual")
	if err != nil {
		writeError(c, err)
		return
	}
	protocol.OK(c, dto.Run(*run, false))
}

func (h *Handler) listRuns(c *gin.Context) {
	var req runListRequest
	if !h.bind(c, &req) {
		return
	}
	filters := dropEmpty(map[string]any{"status": req.Status})
	if req.PipelineID > 0 {
		filters["pipelineId"] = req.PipelineID
	}
	h.search(c, query.PermRunView, "pageForRun", req.Page, req.Size, filters)
}

func (h *Handler) getRun(c *gin.Context) {
	var req idRequest
	if !h.bind(c, &req) || !h.requireID(c, req.ID) || !h.allow(c, query.PermRunView) {
		return
	}
	item, err := h.pipelines.GetRun(req.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	protocol.OK(c, dto.Run(*item, true))
}

func (h *Handler) listDatasets(c *gin.Context) {
	var req datasetListRequest
	if !h.bind(c, &req) {
		return
	}
	h.search(c, query.PermDatasetView, "pageForDataset", req.Page, req.Size, dropEmpty(map[string]any{
		"name": req.Name,
		"kind": req.Kind,
	}))
}

func (h *Handler) previewDataset(c *gin.Context) {
	var req datasetNameRequest
	if !h.bind(c, &req) || !h.requireName(c, req.Name) || !h.allow(c, query.PermDatasetView) {
		return
	}
	view, err := h.datasets.Preview(req.Name, req.Limit)
	if err != nil {
		writeError(c, err)
		return
	}
	protocol.OK(c, view)
}

func (h *Handler) exportDataset(c *gin.Context) {
	var req datasetNameRequest
	if !h.bind(c, &req) || !h.requireName(c, req.Name) || !h.allow(c, query.PermDatasetEdit) {
		return
	}
	filename, body, err := h.datasets.Export(req.Name, req.Format)
	if err != nil {
		writeError(c, err)
		return
	}
	contentType := "text/csv; charset=utf-8"
	if strings.EqualFold(req.Format, "json") {
		contentType = "application/json; charset=utf-8"
	}
	protocol.OK(c, gin.H{
		"fileName":    filename,
		"contentType": contentType,
		"content":     string(body),
	})
}

func (h *Handler) deleteDataset(c *gin.Context) {
	var req datasetNameRequest
	if !h.bind(c, &req) || !h.requireName(c, req.Name) || !h.allow(c, query.PermDatasetEdit) {
		return
	}
	if err := h.datasets.Delete(req.Name); err != nil {
		writeError(c, err)
		return
	}
	protocol.OK(c, nil)
}

func (h *Handler) uploadFile(c *gin.Context) {
	if !h.allow(c, query.PermFileUpload) {
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		protocol.Fail(c, protocol.CodeBadRequest, "请选择文件")
		return
	}
	defer file.Close()
	item, err := h.sources.SaveUpload(c.PostForm("businessId"), c.PostForm("businessType"), file, header.Filename)
	if err != nil {
		writeError(c, err)
		return
	}
	protocol.OK(c, dto.Source(*item))
}

func (h *Handler) querySearch(c *gin.Context) {
	h.runQuery(c, query.ModeSearch)
}

func (h *Handler) queryList(c *gin.Context) {
	h.runQuery(c, query.ModeList)
}

func (h *Handler) queryOne(c *gin.Context) {
	h.runQuery(c, query.ModeOne)
}

func (h *Handler) runQuery(c *gin.Context, mode string) {
	var req query.Request
	if !h.bind(c, &req) {
		return
	}
	out, err := h.queries.Execute(c.Request.Context(), h.principal(c), mode, req)
	if err != nil {
		writeError(c, err)
		return
	}
	if out.ResponseType == "page" {
		protocol.Page(c, out.Data, out.Page, out.Size, out.Total)
		return
	}
	protocol.OK(c, out.Data)
}

func (h *Handler) search(c *gin.Context, permission, modelName string, page, size int, filters map[string]any) {
	if !h.allow(c, permission) {
		return
	}
	out, err := h.queries.Execute(c.Request.Context(), h.principal(c), query.ModeSearch, query.Request{
		Model:   modelName,
		Page:    page,
		Size:    size,
		Filters: filters,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	protocol.Page(c, out.Data, out.Page, out.Size, out.Total)
}

func (h *Handler) one(c *gin.Context, permission, modelName string, filters map[string]any) {
	if !h.allow(c, permission) {
		return
	}
	out, err := h.queries.Execute(c.Request.Context(), h.principal(c), query.ModeOne, query.Request{
		Model:   modelName,
		Filters: filters,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	protocol.OK(c, out.Data)
}

func (h *Handler) authenticate(c *gin.Context) {
	header := c.GetHeader("Authorization")
	token := strings.TrimPrefix(header, "Bearer ")
	if h.token == "" || !strings.HasPrefix(header, "Bearer ") || subtle.ConstantTimeCompare([]byte(token), []byte(h.token)) != 1 {
		protocol.Fail(c, protocol.CodeUnauthorized, "未认证")
		c.Abort()
		return
	}
	c.Set("principal", query.FullAccess())
	c.Next()
}

func (h *Handler) rejectTenant(c *gin.Context) {
	if !strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "application/json") {
		c.Next()
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		protocol.Fail(c, protocol.CodeBadRequest, "请求体不是合法 JSON")
		c.Abort()
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	if len(bytes.TrimSpace(body)) == 0 {
		c.Next()
		return
	}
	var payload map[string]json.RawMessage
	if json.Unmarshal(body, &payload) != nil {
		c.Next()
		return
	}
	if _, ok := payload["tenantId"]; ok {
		protocol.Fail(c, protocol.CodeBadRequest, "不能指定租户")
		c.Abort()
		return
	}
	c.Next()
}

func (h *Handler) bind(c *gin.Context, dest any) bool {
	if err := c.ShouldBindJSON(dest); err != nil {
		protocol.Fail(c, protocol.CodeBadRequest, "请求体不是合法 JSON")
		return false
	}
	return true
}

func (h *Handler) allow(c *gin.Context, permission string) bool {
	if h.principal(c).Allow(permission) {
		return true
	}
	protocol.Fail(c, protocol.CodeForbidden, "无权限")
	return false
}

func (h *Handler) principal(c *gin.Context) query.Principal {
	value, _ := c.Get("principal")
	principal, _ := value.(query.Principal)
	return principal
}

func (h *Handler) requireID(c *gin.Context, id uint) bool {
	if id > 0 {
		return true
	}
	protocol.Fail(c, protocol.CodeBadRequest, "id 不能为空")
	return false
}

func (h *Handler) requireName(c *gin.Context, name string) bool {
	if strings.TrimSpace(name) != "" {
		return true
	}
	protocol.Fail(c, protocol.CodeBadRequest, "name 不能为空")
	return false
}

func toPipelineInput(req pipelineWriteRequest) service.PipelineInput {
	return service.PipelineInput{
		Name:            req.Name,
		Description:     req.Description,
		Enabled:         req.Enabled,
		SourceID:        req.SourceID,
		Steps:           req.Steps,
		TargetType:      req.TargetType,
		TargetConfig:    req.TargetConfig,
		ScheduleMinutes: req.ScheduleMinutes,
	}
}

func dropEmpty(filters map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range filters {
		text, ok := value.(string)
		if ok && strings.TrimSpace(text) == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func writeError(c *gin.Context, err error) {
	var pe *protocol.Error
	if errors.As(err, &pe) {
		protocol.Fail(c, pe.Code, pe.Msg)
		return
	}
	var se *service.Error
	if errors.As(err, &se) {
		protocol.Fail(c, se.Code, se.Msg)
		return
	}
	if err == io.EOF {
		protocol.Fail(c, protocol.CodeBadRequest, "请求体为空")
		return
	}
	msg := err.Error()
	if strings.Contains(msg, "UNIQUE constraint failed") {
		protocol.Fail(c, protocol.CodeConflict, "名称已存在")
		return
	}
	protocol.Fail(c, protocol.CodeError, msg)
}
