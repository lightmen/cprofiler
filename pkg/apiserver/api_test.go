package apiserver

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"cprofiler/pkg/storage"
	"cprofiler/pkg/storage/badger"

	"github.com/gavv/httpexpect/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

var (
	profileMetas = []*storage.ProfileMeta{
		{
			ProfileID:      "1",
			Timestamp:      time.Now().UnixNano() / time.Millisecond.Nanoseconds(),
			Duration:       time.Now().UnixNano(),
			SampleTypeUnit: "count",
			SampleType:     "heap_alloc_objects",
			ProfileType:    "heap",
			JobName:        "profiler-server",
			Value:          100,
		},
		{
			ProfileID:      "2",
			Timestamp:      time.Now().UnixNano() / time.Millisecond.Nanoseconds(),
			Duration:       time.Now().UnixNano(),
			SampleTypeUnit: "bytes",
			SampleType:     "heap_alloc_space",
			ProfileType:    "heap",
			JobName:        "profiler-server",
			Value:          200,
		},
		{
			ProfileID:      "3",
			Timestamp:      time.Now().UnixNano() / time.Millisecond.Nanoseconds(),
			Duration:       time.Now().UnixNano(),
			SampleTypeUnit: "count",
			SampleType:     "heap_inuse_objects",
			ProfileType:    "heap",
			JobName:        "server2",
			Value:          300,
		},
		{
			ProfileID:      "4",
			Timestamp:      time.Now().UnixNano() / time.Millisecond.Nanoseconds(),
			Duration:       time.Now().UnixNano(),
			SampleTypeUnit: "bytes",
			SampleType:     "heap_inuse_space",
			ProfileType:    "heap",
			JobName:        "server2",
			Value:          400,
		},
		{
			ProfileID:      "5",
			Timestamp:      time.Now().UnixNano() / time.Millisecond.Nanoseconds(),
			Duration:       time.Now().UnixNano(),
			SampleTypeUnit: "bytes",
			SampleType:     "heap_inuse_space",
			ProfileType:    "heap",
			JobName:        "server3",
			Value:          400,
		},
	}
)

// Rid of debug output
func init() {
	gin.SetMode(gin.TestMode)
}

func initMateData(s storage.Store, t *testing.T) {
	err := s.SaveProfileMeta(profileMetas, time.Hour)
	require.Equal(t, nil, err)
}

func initProfileData(s storage.Store, t *testing.T) (string, string, string, string) {
	invalidId, err := s.SaveProfile("", []byte{}, time.Hour)
	require.Equal(t, nil, err)

	invalidId2, err := s.SaveProfile("", []byte("haha"), time.Hour)
	require.Equal(t, nil, err)

	profileBytes, err := ioutil.ReadFile("./testdata/profile.out.testdata")
	require.Equal(t, nil, err)
	id, err := s.SaveProfile("", profileBytes, time.Hour)
	require.Equal(t, nil, err)

	traceBytes, err := ioutil.ReadFile("./testdata/trace.out.testdata")
	require.Equal(t, nil, err)
	traceID, err := s.SaveProfile("", traceBytes, time.Hour)
	require.Equal(t, nil, err)
	return invalidId, invalidId2, id, traceID
}

func TestApiServer(t *testing.T) {
	dir, err := ioutil.TempDir("./", "temp-*")
	require.Equal(t, nil, err)
	defer os.RemoveAll(dir)
	s := badger.NewStore(badger.DefaultOptions(dir))
	apiServer := NewAPIServer(DefaultOptions(s))
	apiServer.Run()
	defer apiServer.Stop()
}

func TestBasisAPI(t *testing.T) {
	dir, err := ioutil.TempDir("./", "temp-*")
	require.Equal(t, nil, err)
	defer os.RemoveAll(dir)
	s := badger.NewStore(badger.DefaultOptions(dir))

	initMateData(s, t)

	apiServer := NewAPIServer(DefaultOptions(s))

	e := getExpect(apiServer, t)

	e.GET("/").
		Expect().
		Status(http.StatusNotFound)

	e.GET("/api/healthz").
		Expect().
		Status(http.StatusOK)

	e.GET("/api/targets").
		Expect().
		Status(http.StatusOK).JSON().Array().Contains("profiler-server", "server2", "server3")

	e.GET("/api/sample_types").
		Expect().
		Status(http.StatusOK).JSON().Array().Contains("heap_alloc_objects", "heap_alloc_space", "heap_inuse_space", "heap_inuse_space")

	res := e.GET("/api/group_sample_types").
		Expect().
		Status(http.StatusOK).JSON().Object()
	res.Path("$").Object().Keys().Length().Equal(1)
	res.Path("$.heap").Array().Contains("heap_alloc_objects", "heap_alloc_space", "heap_inuse_space", "heap_inuse_space")
}

func TestListProfileMeta(t *testing.T) {
	dir, err := ioutil.TempDir("./", "temp-*")
	require.Equal(t, nil, err)
	defer os.RemoveAll(dir)

	s := badger.NewStore(badger.DefaultOptions(dir))

	initMateData(s, t)

	apiServer := NewAPIServer(DefaultOptions(s))

	e := getExpect(apiServer, t)

	e.GET("/api/profile_meta").
		Expect().
		Status(http.StatusNotFound)

	e.GET("/api/profile_meta/heap_inuse_space").
		Expect().
		Status(http.StatusBadRequest).Text().Equal("start_time or end_time is empty")

	startTimestamp := time.Now().Add(-1*time.Minute).UnixNano() / time.Millisecond.Nanoseconds()
	endTimestamp := time.Now().UnixNano() / time.Millisecond.Nanoseconds()

	e.GET("/api/profile_meta/heap_inuse_space").WithQuery("start_time", startTimestamp).
		Expect().
		Status(http.StatusBadRequest).Text().Equal("start_time or end_time is empty")

	e.GET("/api/profile_meta/heap_inuse_space").
		WithQuery("start_time", startTimestamp).WithQuery("end_time", endTimestamp).
		Expect().
		Status(http.StatusBadRequest).Text().Contains("The time format must be RFC3339")

	startTime := time.Now().Add(-1 * time.Minute).Format(time.RFC3339)
	endTime := time.Now().Format(time.RFC3339)

	e.GET("/api/profile_meta/heap_inuse_space").
		WithQuery("start_time", startTime).WithQuery("end_time", endTimestamp).
		Expect().
		Status(http.StatusBadRequest).Text().Contains("The time format must be RFC3339")

	e.GET("/api/profile_meta/heap_inuse_space").
		WithQuery("start_time", startTime).WithQuery("end_time", endTime).
		Expect().
		Status(http.StatusOK).JSON().Array().Length().Equal(0)
}

func TestDownloadProfile(t *testing.T) {
	dir, err := ioutil.TempDir("./", "temp-*")
	require.Equal(t, nil, err)
	defer os.RemoveAll(dir)

	s := badger.NewStore(badger.DefaultOptions(dir))
	_, _, id, traceID := initProfileData(s, t)
	apiServer := NewAPIServer(DefaultOptions(s))
	e := getExpect(apiServer, t)

	e.GET("/api/download/999").
		Expect().
		Status(http.StatusNotFound)

	e.GET(fmt.Sprintf("/api/download/%s", id)).
		Expect().
		Status(http.StatusOK).Header("Content-Type").Equal("application/octet-stream")

	e.GET("/api/trace/999").
		Expect().
		Status(http.StatusNotFound)

	e.GET(fmt.Sprintf("/api/download/%s", traceID)).
		Expect().
		Status(http.StatusOK).Header("Content-Type").Equal("application/octet-stream")
}

func TestWebProfile(t *testing.T) {
	dir, err := ioutil.TempDir("./", "temp-*")
	require.Equal(t, nil, err)
	defer os.RemoveAll(dir)

	s := badger.NewStore(badger.DefaultOptions(dir))
	apiServer := NewAPIServer(DefaultOptions(s))
	e := getExpect(apiServer, t)
	e.GET("/api/pprof/ui/1999").
		Expect().
		Status(http.StatusNotFound).Text().Equal("Profile not found\n")
}

func TestTraceProfile(t *testing.T) {
	dir, err := ioutil.TempDir("./", "temp-*")
	require.Equal(t, nil, err)
	defer os.RemoveAll(dir)

	s := badger.NewStore(badger.DefaultOptions(dir))
	apiServer := NewAPIServer(DefaultOptions(s))
	e := getExpect(apiServer, t)
	e.GET("/api/trace/ui/1999").
		Expect().
		Status(http.StatusNotFound).Text().Equal("Profile not found\n")
}

func getExpect(apiServer *APIServer, t *testing.T) *httpexpect.Expect {
	handler := apiServer.router

	return httpexpect.WithConfig(httpexpect.Config{
		Client: &http.Client{
			Transport: httpexpect.NewBinder(handler),
			Jar:       httpexpect.NewJar(),
		},
		Reporter: httpexpect.NewAssertReporter(t),
	})
}
