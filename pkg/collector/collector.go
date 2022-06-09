package collector

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"sync"
	"time"

	"cprofiler/pkg/storage"

	"github.com/google/pprof/profile"
	"github.com/sirupsen/logrus"
)

// Collector Collect target pprof http endpoints
type Collector struct {
	JobName string
	ScrapeConfig
	Profiles map[string]*ProfileConfig
	Target   TargetConfig
	Host     string

	exitChan        chan struct{}
	resetTickerChan chan time.Duration
	mangerWg        *sync.WaitGroup
	wg              *sync.WaitGroup
	httpClient      *http.Client
	mu              sync.RWMutex
	log             *logrus.Entry
	store           storage.Store
}

func newCollector(job JobConfig, store storage.Store, mangerWg *sync.WaitGroup) *Collector {
	collector := &Collector{
		JobName:      job.Scrape.Job,
		ScrapeConfig: *job.Scrape,
		Host:         job.Host,
		Target:       *job.Target,
		Profiles:     make(map[string]*ProfileConfig),

		exitChan:        make(chan struct{}),
		resetTickerChan: make(chan time.Duration, 1000),
		mangerWg:        mangerWg,
		wg:              &sync.WaitGroup{},
		httpClient:      &http.Client{},
		log:             logrus.WithField("collector", job.Scrape.Job),
		store:           store,
	}

	collector.Profiles = buildProfileConfigs(&collector.ScrapeConfig)
	return collector
}

func (collector *Collector) run() {
	collector.mu.Lock()
	defer collector.mu.Unlock()

	collector.log.Info("collector run")

	collector.mangerWg.Add(1)

	go collector.scrapeLoop(collector.Interval)
}

func (collector *Collector) scrapeLoop(interval time.Duration) {
	defer collector.mangerWg.Done()
	collector.scrape()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-collector.exitChan:
			collector.log.Info("scrape loop exit")
			return
		case i := <-collector.resetTickerChan:
			ticker.Reset(i)
		case <-ticker.C:
			collector.scrape()
		}
	}
}

func (collector *Collector) reload(host string, job JobConfig) {
	collector.mu.Lock()
	defer collector.mu.Unlock()

	collector.Profiles = buildProfileConfigs(&collector.ScrapeConfig)

	if reflect.DeepEqual(collector.ScrapeConfig, job) {
		return
	}
	collector.log.Info("reload collector ")

	scrape := job.Scrape
	if collector.Interval != scrape.Interval {
		collector.resetTickerChan <- scrape.Interval
	}

	collector.ScrapeConfig = *job.Scrape
	collector.Target = *job.Target
	collector.Host = job.Host
}

func (collector *Collector) exit() {
	close(collector.exitChan)
}

func (collector *Collector) scrape() {
	collector.mu.RLock()
	defer collector.mu.RUnlock()

	collector.log.Info("collector start scrape")
	for profileType, profileConfig := range collector.Profiles {
		if profileConfig.Enable {
			collector.wg.Add(1)
			go collector.fetch(profileType, profileConfig)
		}
	}
	collector.wg.Wait()
}

func (collector *Collector) fetch(profileType string, profileConfig *ProfileConfig) {
	defer collector.wg.Done()

	logEntry := collector.log.WithFields(logrus.Fields{"profile_type": profileType, "profile_url": profileConfig.Path})
	logEntry.Info("collector start fetch")

	req, err := http.NewRequest("GET", "http://"+collector.Host+profileConfig.Path, nil)
	if err != nil {
		logEntry.WithError(err).Error("invoke task error")
		return
	}
	req.Header.Set("User-Agent", "")

	resp, err := collector.httpClient.Do(req)
	if err != nil {
		logEntry.WithError(err).Error("http request error")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logEntry.WithError(err).Error("http resp status code is ", resp.StatusCode)
		return
	}

	profileBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logEntry.WithError(err).Error("read resp error")
		return
	}

	if profileType == "trace" {
		err = collector.analysisTrace(profileType, profileBytes)
		if err != nil {
			logEntry.WithError(err).Error("analysis result error")
			return
		}
		return
	}

	err = collector.analysis(profileType, profileBytes)
	if err != nil {
		logEntry.WithError(err).Error("analysis result error")
		return
	}
}

func (collector *Collector) analysis(profileType string, profileBytes []byte) error {
	p, err := profile.ParseData(profileBytes)
	if err != nil {
		return err
	}
	if len(p.SampleType) == 0 {
		return errors.New("sample type is nil")
	}

	// Set profile name , Display it on the Profile UI
	if len(p.Mapping) > 0 {
		p.Mapping[0].File = collector.JobName
	}

	b := &bytes.Buffer{}
	if err = p.Write(b); err != nil {
		return err
	}

	profileID, err := collector.store.SaveProfile(fmt.Sprintf("%s-%s", collector.JobName, profileType), b.Bytes(), collector.Expiration)
	if err != nil {
		return err
	}

	metas := make([]*storage.ProfileMeta, 0, len(p.SampleType))
	for i := range p.SampleType {
		meta := &storage.ProfileMeta{}
		meta.Timestamp = time.Now().UnixNano() / time.Millisecond.Nanoseconds()
		meta.ProfileID = profileID
		meta.ProfileType = profileType
		meta.JobName = collector.JobName
		meta.Host = collector.Host
		meta.App = collector.Target.Application

		meta.Duration = p.DurationNanos
		meta.SampleTypeUnit = p.SampleType[i].Unit
		for _, s := range p.Sample {
			meta.Value += s.Value[i]
		}
		if len(p.SampleType) > 1 {
			meta.SampleType = fmt.Sprintf("%s_%s", profileType, p.SampleType[i].Type)
		} else {
			meta.SampleType = profileType
		}

		meta.Labels = collector.Target.Labels.ToArray()
		metas = append(metas, meta)
	}

	err = collector.store.SaveProfileMeta(metas, collector.Expiration)
	if err != nil {
		return err
	}
	return nil
}

func (collector *Collector) analysisTrace(profileType string, profileBytes []byte) error {
	profileID, err := collector.store.SaveProfile(fmt.Sprintf("%s-%s", collector.JobName, profileType), profileBytes, collector.Expiration)
	if err != nil {
		return err
	}

	metas := make([]*storage.ProfileMeta, 0, 1)
	meta := &storage.ProfileMeta{}
	meta.Timestamp = time.Now().UnixNano() / time.Millisecond.Nanoseconds()
	meta.ProfileID = profileID
	meta.ProfileType = profileType
	meta.SampleType = profileType
	meta.JobName = collector.JobName
	meta.Host = collector.Host
	meta.App = collector.Target.Application

	meta.Labels = collector.Target.Labels.ToArray()
	metas = append(metas, meta)

	err = collector.store.SaveProfileMeta(metas, collector.Expiration)
	if err != nil {
		return err
	}
	return nil
}
