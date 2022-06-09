package collector

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	generalConfigYAML = `
  scrape-configs:
  - job: test
    interval: 60s         # the interval for scrape profile data
    expiration: 168h      # the profile data expire time
    # enabled-profiles include: profile, mutex, heap, goroutine, allocs, block, threadcreate, trace , all
    # all : means all sample type except trace
    # if not
    enabled-profiles: [all, trace]
    path-profiles: 
      profile: /debug/pprof/profile?seconds=10
      trace: /debug/pprof/trace?seconds=10
    target-configs:
      - application: lobbysrv
        hosts:
          - 192.168.15.115:8150
        labels:                
          env: dev
      - application: pokersrv
        hosts:
          - 192.168.15.115:16012 
          - 192.168.15.115:16022 
          - 192.168.15.115:9656 
        labels:              
          env: dev
      - application: gatesrv
        hosts:
          - 192.168.15.115:16002
        labels:              
          env: dev 
`

	changeConfigYAML = `
  scrape-configs:
  - job: change
    interval: 60s         # the interval for scrape profile data
    expiration: 168h      # the profile data expire time
    # enabled-profiles include: profile, mutex, heap, goroutine, allocs, block, threadcreate, trace , all
    # all : means all sample type except trace
    # if not
    enabled-profiles: [all, trace]
    path-profiles: 
      profile: /debug/pprof/profile?seconds=10
      trace: /debug/pprof/trace?seconds=10
    target-configs:
      - application: lobbysrv
        hosts:
          - 192.168.15.115:8150
        labels:                
          env: dev
      - application: pokersrv
        hosts:
          - 192.168.15.115:16012 
          - 192.168.15.115:16022 
          - 192.168.15.115:9656 
        labels:              
          env: dev
      - application: gatesrv
        hosts:
          - 192.168.15.115:16002
        labels:              
          env: dev 
`
)

func TestChangeConfig(t *testing.T) {
	file, err := ioutil.TempFile("./", "config-*.yml")
	require.Equal(t, err, nil)
	defer func() {
		name := file.Name()
		file.Close()
		os.Remove(name)
	}()

	_, err = file.Write([]byte(generalConfigYAML))
	file.Sync()

	require.Equal(t, err, nil)
	flag := 0

	err = LoadConfig(file.Name(), func(config CollectorConfig) {
		if flag == 0 {
			require.NotEqual(t, nil, config)
			require.Equal(t, true, len(config.ScrapeConfigs) >= 1)
			require.Equal(t, "test", config.ScrapeConfigs[0].Job)
		} else {
			require.NotEqual(t, nil, config)
			require.Equal(t, true, len(config.ScrapeConfigs) >= 1)
			require.Equal(t, "change", config.ScrapeConfigs[0].Job)
		}
		flag++
	})

	_, err = file.Write([]byte(changeConfigYAML))
	file.Sync()

	require.Equal(t, nil, err)
	time.Sleep(1 * time.Second)
	require.Equal(t, 2, flag)
}

func TestLoadConfig(t *testing.T) {
	file, err := ioutil.TempFile("./", "config-*.yml")
	require.Equal(t, nil, err)
	defer func() {
		name := file.Name()
		file.Close()
		os.Remove(name)
	}()
	_, err = file.Write([]byte(generalConfigYAML))
	require.Equal(t, nil, err)

	err = LoadConfig(file.Name(), func(config CollectorConfig) {
		require.NotEqual(t, config, nil)
		require.Equal(t, true, len(config.ScrapeConfigs) >= 1)

		scrape := config.ScrapeConfigs[0]
		require.Equal(t, 60*time.Second, scrape.Interval)

		js, _ := json.Marshal(config)
		t.Logf("js: %s", string(js))
	})

	require.Equal(t, nil, err)
}
