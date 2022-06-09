package collector

import (
	"io/ioutil"
	"log"
	"time"

	"cprofiler/pkg/storage"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v2"
)

func getConfig(fileName string) (config CollectorConfig, err error) {
	buf, err := ioutil.ReadFile(fileName)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(buf, &config)
	if err != nil {
		return
	}

	return
}

// LoadConfig watch configPath change, callback fn
func LoadConfig(configPath string, fn func(CollectorConfig)) error {
	config, err := getConfig(configPath)
	if err != nil {
		log.Printf("getConfig %s error: %s", configPath, err.Error())
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf(" fsnotify.NewWatcher error: %s", err.Error())
		return err
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				// const writeOrCreateMask = fsnotify.Write | fsnotify.Create
				// if event.Op&writeOrCreateMask == 0 {
				// 	continue
				// }

				log.Printf("file %s changed, event: %s", event.Name, event.Op.String())

				var newConfig CollectorConfig
				newConfig, err1 := getConfig(event.Name)
				if err1 != nil {
					log.Printf("getConfig %s error: %s", event.Name, err1.Error())
					continue
				}

				fn(newConfig)
			case err = <-watcher.Errors:
				log.Printf("watcher error: %s", err.Error())
			}
		}
	}()

	err = watcher.Add(configPath)
	if err != nil {
		log.Printf("watcher.Add %s error: %s", configPath, err.Error())
		return err
	}

	fn(config)

	return nil
}

type CollectorConfig struct {
	ScrapeConfigs []ScrapeConfig `yaml:"scrape-configs"`
}

type ScrapeConfig struct {
	Job             string            `yaml:"job"`
	Interval        time.Duration     `yaml:"interval"`
	Expiration      time.Duration     `yaml:"expiration"`
	EnabledProfiles []string          `yaml:"enabled-profiles"`
	Path            map[string]string `yaml:"path-profiles"`
	Targets         []TargetConfig    `yaml:"target-configs"`
}

type TargetConfig struct {
	Application string      `yaml:"application"`
	Hosts       []string    `yaml:"hosts"`
	Labels      LabelConfig `yaml:"labels"`
}

type ProfileConfig struct {
	Path   string
	Enable bool
}

type JobConfig struct {
	Scrape *ScrapeConfig
	Target *TargetConfig
	Host   string
}

type LabelConfig map[string]string

func (t LabelConfig) ToArray() []storage.Label {
	labels := make([]storage.Label, 0, len(t))
	for k, v := range t {
		labels = append(labels, storage.Label{
			Key:   k,
			Value: v,
		})
	}
	return labels
}

// defaultProfileConfigs The default fetching profile config
func defaultProfileConfigs() map[string]*ProfileConfig {
	return map[string]*ProfileConfig{
		"profile": {
			Path: "/debug/pprof/profile?seconds=10",
		},
		"mutex": {
			Path: "/debug/pprof/mutex",
		},
		"heap": {
			Path: "/debug/pprof/heap",
		},
		"goroutine": {
			Path: "/debug/pprof/goroutine",
		},
		"allocs": {
			Path: "/debug/pprof/allocs",
		},
		"block": {
			Path: "/debug/pprof/block",
		},
		"threadcreate": {
			Path: "/debug/pprof/threadcreate",
		},
		"trace": {
			Path: "/debug/pprof/trace?seconds=10",
		},
	}
}

func buildProfileConfigs(scrape *ScrapeConfig) map[string]*ProfileConfig {
	cfgs := defaultProfileConfigs()

	enables := scrape.EnabledProfiles
	paths := scrape.Path

	isAll := false
	if len(enables) == 0 {
		isAll = true
	}

	for _, val := range enables {
		if val == "all" {
			isAll = true
			continue
		}

		if _, ok := cfgs[val]; !ok {
			continue
		}

		cfgs[val].Enable = true
	}

	for key, item := range cfgs {
		if isAll {
			item.Enable = true
		}

		if _, ok := paths[key]; ok {
			item.Path = paths[key]
		}
	}

	return cfgs
}
