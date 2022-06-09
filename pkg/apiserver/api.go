package apiserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cprofiler/pkg/apiserver/ui"
	"cprofiler/pkg/apiserver/ui/pprof"
	"cprofiler/pkg/apiserver/ui/trace"
	"cprofiler/pkg/storage"
	"cprofiler/pkg/utils"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type APIServer struct {
	opt    Options
	store  storage.Store
	router *gin.Engine
	srv    *http.Server
	pprof  *ui.Server
	trace  *ui.Server
}

func NewAPIServer(opt Options) *APIServer {
	pprofPath := "/api/pprof/ui"
	tracePath := "/api/trace/ui"

	apiServer := &APIServer{
		opt:   opt,
		store: opt.Store,
		pprof: ui.NewServer(pprofPath, opt.Store, opt.GCInternal, pprof.Driver),
		trace: ui.NewServer(tracePath, opt.Store, opt.GCInternal, trace.Driver),
	}

	router := gin.Default()
	router.GET("/api/healthz", func(c *gin.Context) {
		c.String(200, "I'm fine")
	})
	router.Use(HandleCors).GET("/api/targets", apiServer.listTarget)
	router.Use(HandleCors).GET("/api/group_labels", apiServer.listGroupLabel)
	router.Use(HandleCors).GET("/api/sample_types", apiServer.listSampleTypes)
	router.Use(HandleCors).GET("/api/group_sample_types", apiServer.listGroupSampleTypes)
	router.Use(HandleCors).GET("/api/profile_meta/:sample_type", apiServer.listProfileMeta)
	router.Use(HandleCors).GET("/api/download/:id", apiServer.downloadProfile)

	// register pprof page
	router.Use(HandleCors).GET(pprofPath+"/*any", apiServer.webPProf)
	// register trace page
	router.Use(HandleCors).GET(tracePath+"/*any", apiServer.webTrace)

	srv := &http.Server{
		Addr:    opt.Addr,
		Handler: router,
	}

	apiServer.router = router
	apiServer.srv = srv
	return apiServer
}

func (s *APIServer) Stop() {
	s.pprof.Exit()
	s.trace.Exit()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.srv.Shutdown(ctx); err != nil {
		log.Fatal("api server forced to shutdown:", err)
	}
	log.Info("api server exit ")
}

func (s *APIServer) Run() {
	go func() {
		if err := s.srv.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				log.Info("api server close")
				return
			}
			log.Fatal("api server listen: ", err)
		}
	}()
}

func (s *APIServer) listTarget(c *gin.Context) {
	targets, err := s.store.ListTarget()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, targets)
}

func (s *APIServer) listGroupLabel(c *gin.Context) {
	labels, err := s.store.ListLabel()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	groupLabels := make(map[string][]storage.Label)
	for _, label := range labels {
		if _, ok := groupLabels[label.Key]; !ok {
			groupLabels[label.Key] = make([]storage.Label, 0, 5)
		}
		groupLabels[label.Key] = append(groupLabels[label.Key], label)
	}

	c.JSON(http.StatusOK, groupLabels)
}

func (s *APIServer) listSampleTypes(c *gin.Context) {
	sampleTypes, err := s.store.ListSampleType()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, sampleTypes)
}

func (s *APIServer) listGroupSampleTypes(c *gin.Context) {
	sampleTypes, err := s.store.ListSampleType()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	groupSampleTypes := make(map[string][]string)
	for _, sampleType := range sampleTypes {
		gSampleType := strings.Split(sampleType, "_")[0]
		if _, ok := groupSampleTypes[gSampleType]; !ok {
			groupSampleTypes[gSampleType] = make([]string, 0, 5)
		}
		groupSampleTypes[gSampleType] = append(groupSampleTypes[gSampleType], sampleType)
	}

	c.JSON(http.StatusOK, groupSampleTypes)
}

func (s *APIServer) listProfileMeta(c *gin.Context) {
	var startTime, endTime time.Time
	var err error

	sampleType := c.Param("sample_type")

	if c.Query("start_time") == "" || c.Query("end_time") == "" {
		c.String(http.StatusBadRequest, "start_time or end_time is empty")
		return
	}

	startTime, err = time.Parse("2006-01-02 15:04:05", c.Query("start_time"))
	if err != nil {
		if startTime, err = time.Parse(time.RFC3339, c.Query("start_time")); err != nil {
			c.String(http.StatusBadRequest, "%s ,%s", "The time format must be RFC3339", err.Error())
			return
		}
	}

	endTime, err = time.Parse("2006-01-02 15:04:05", c.Query("end_time"))
	if err != nil {
		if endTime, err = time.Parse(time.RFC3339, c.Query("end_time")); err != nil {
			c.String(http.StatusBadRequest, "%s ,%s", "The time format must be RFC3339", err.Error())
			return
		}
	}

	req := struct {
		Filters []storage.LabelFilter `json:"labels[]" form:"labels[]"`
	}{}

	lbs := c.QueryMap("lbs")

	if len(lbs) > 0 {
		for key, val := range lbs {
			filter := &storage.LabelFilter{
				Label: storage.Label{
					Key:   key,
					Value: val,
				},
				Condition: c.Query("condition"),
			}
			req.Filters = append(req.Filters, *filter)
		}
	} else {
		if err := c.ShouldBind(&req); err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
	}

	metas, err := s.store.ListProfileMeta(sampleType, startTime, endTime, req.Filters...)

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, metas)
}

func (s *APIServer) downloadProfile(c *gin.Context) {
	id := c.Param("id")
	name, data, err := s.store.GetProfile(id)
	if err != nil {
		if errors.Is(err, storage.ErrProfileNotFound) {
			c.String(http.StatusNotFound, "Profile not found")
			return
		}
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%s-%s.prof", name, id))
	c.Data(200, "application/octet-stream", data)
}

func (s *APIServer) webPProf(c *gin.Context) {
	c.Request.URL.RawQuery = utils.RemovePrefixSampleType(c.Request.URL.RawQuery)
	s.pprof.Web(c.Writer, c.Request)
}

func (s *APIServer) webTrace(c *gin.Context) {
	c.Request.URL.RawQuery = utils.RemovePrefixSampleType(c.Request.URL.RawQuery)
	s.trace.Web(c.Writer, c.Request)
}
