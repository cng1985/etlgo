package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"etlgo/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	sources   *service.SourceService
	pipelines *service.PipelineService
	datasets  *service.DatasetService
}

func New(sources *service.SourceService, pipelines *service.PipelineService, datasets *service.DatasetService) *Handler {
	return &Handler{sources: sources, pipelines: pipelines, datasets: datasets}
}

func (h *Handler) Register(r *gin.Engine) {
	api := r.Group("/api")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	api.GET("/stats", h.stats)

	api.GET("/sources", h.listSources)
	api.POST("/sources", h.createSource)
	api.POST("/uploads", h.uploadSource)
	api.GET("/sources/:id", h.getSource)
	api.PUT("/sources/:id", h.updateSource)
	api.DELETE("/sources/:id", h.deleteSource)
	api.POST("/sources/:id/preview", h.previewSource)

	api.GET("/pipelines", h.listPipelines)
	api.POST("/pipelines", h.createPipeline)
	api.POST("/pipelines/preview", h.previewPipeline)
	api.GET("/pipelines/:id", h.getPipeline)
	api.PUT("/pipelines/:id", h.updatePipeline)
	api.DELETE("/pipelines/:id", h.deletePipeline)
	api.POST("/pipelines/:id/run", h.runPipeline)

	api.GET("/runs", h.listRuns)
	api.GET("/runs/:id", h.getRun)

	api.GET("/datasets", h.listDatasets)
	api.GET("/datasets/:name/preview", h.previewDataset)
	api.GET("/datasets/:name/export", h.exportDataset)
	api.DELETE("/datasets/:name", h.deleteDataset)
}

func (h *Handler) stats(c *gin.Context) {
	stats, err := h.datasets.Stats()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) listSources(c *gin.Context) {
	items, err := h.sources.List()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) createSource(c *gin.Context) {
	var in service.SourceInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体不是合法 JSON"})
		return
	}
	item, err := h.sources.Create(in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) uploadSource(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择文件"})
		return
	}
	defer file.Close()
	item, err := h.sources.SaveUpload(c.PostForm("name"), c.PostForm("type"), file, header.Filename)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) getSource(c *gin.Context) {
	item, err := h.sources.Get(uintParam(c, "id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) updateSource(c *gin.Context) {
	var in service.SourceInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体不是合法 JSON"})
		return
	}
	item, err := h.sources.Update(uintParam(c, "id"), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) deleteSource(c *gin.Context) {
	if err := h.sources.Delete(uintParam(c, "id")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) previewSource(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	view, err := h.sources.Preview(c.Request.Context(), uintParam(c, "id"), limit)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *Handler) listPipelines(c *gin.Context) {
	items, err := h.pipelines.List()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) createPipeline(c *gin.Context) {
	var in service.PipelineInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体不是合法 JSON"})
		return
	}
	item, err := h.pipelines.Create(in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) getPipeline(c *gin.Context) {
	item, err := h.pipelines.Get(uintParam(c, "id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) updatePipeline(c *gin.Context) {
	var in service.PipelineInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体不是合法 JSON"})
		return
	}
	item, err := h.pipelines.Update(uintParam(c, "id"), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) deletePipeline(c *gin.Context) {
	if err := h.pipelines.Delete(uintParam(c, "id")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) previewPipeline(c *gin.Context) {
	var in service.PreviewInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体不是合法 JSON"})
		return
	}
	view, err := h.pipelines.Preview(c.Request.Context(), in)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *Handler) runPipeline(c *gin.Context) {
	run, err := h.pipelines.StartRun(uintParam(c, "id"), "manual")
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, run)
}

func (h *Handler) listRuns(c *gin.Context) {
	pipelineID, _ := strconv.Atoi(c.Query("pipelineId"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	items, err := h.pipelines.ListRuns(uint(pipelineID), c.Query("status"), limit)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) getRun(c *gin.Context) {
	item, err := h.pipelines.GetRun(uintParam(c, "id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) listDatasets(c *gin.Context) {
	items, err := h.datasets.List()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) previewDataset(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	view, err := h.datasets.Preview(c.Param("name"), limit)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *Handler) exportDataset(c *gin.Context) {
	format := c.DefaultQuery("format", "csv")
	filename, body, err := h.datasets.Export(c.Param("name"), format)
	if err != nil {
		writeError(c, err)
		return
	}
	contentType := "text/csv; charset=utf-8"
	if format == "json" {
		contentType = "application/json; charset=utf-8"
	}
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(http.StatusOK, contentType, body)
}

func (h *Handler) deleteDataset(c *gin.Context) {
	if err := h.datasets.Delete(c.Param("name")); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func uintParam(c *gin.Context, name string) uint {
	n, _ := strconv.Atoi(c.Param(name))
	if n < 0 {
		return 0
	}
	return uint(n)
}

func writeError(c *gin.Context, err error) {
	var se *service.Error
	if errors.As(err, &se) {
		c.JSON(se.Code, gin.H{"error": se.Msg})
		return
	}
	if err == io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求体为空"})
		return
	}
	msg := err.Error()
	if strings.Contains(msg, "UNIQUE constraint failed") {
		c.JSON(http.StatusConflict, gin.H{"error": "名称已存在"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
}
