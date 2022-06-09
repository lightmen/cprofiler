package collector

import (
	"sync"

	"cprofiler/pkg/storage"

	log "github.com/sirupsen/logrus"
)

// Manger Manage multiple collectors to scraping
type Manger struct {
	collectors map[string]*Collector
	store      storage.Store
	wg         *sync.WaitGroup
	mu         sync.Mutex
}

// NewManger new Manger instance
func NewManger(store storage.Store) *Manger {
	c := &Manger{
		collectors: make(map[string]*Collector),
		store:      store,
		wg:         &sync.WaitGroup{},
	}
	return c
}

// NewManger stop Manger instance
func (manger *Manger) Stop() {
	manger.mu.Lock()
	defer manger.mu.Unlock()
	for _, c := range manger.collectors {
		c.exit()
	}
	manger.wg.Wait()
	log.Info("collector manger exit ")
}

// NewManger Loading collector configuration
// It can be called multiple times, and the collector updates the configuration
func (manger *Manger) Load(config CollectorConfig) {
	manger.mu.Lock()
	defer manger.mu.Unlock()

	hosts := make(map[string]JobConfig)
	for _, scrape := range config.ScrapeConfigs {
		for _, target := range scrape.Targets {
			for _, host := range target.Hosts {
				_scrape := scrape
				_target := target
				hosts[host] = JobConfig{
					Scrape: &_scrape,
					Target: &_target,
					Host:   host,
				}
			}
		}

	}

	// delete old collector
	for key, collector := range manger.collectors {
		if _, ok := hosts[key]; !ok {
			log.Info("delete collector ", key)
			collector.exit()
			delete(manger.collectors, key)
		}
	}

	for host, job := range hosts {
		collector, ok := manger.collectors[host]
		if !ok {
			// add collector
			log.Info("add collector ", job.Scrape.Job, "_", job.Target.Application, "_", host)
			collector := newCollector(job, manger.store, manger.wg)
			manger.collectors[host] = collector
			collector.run()
			continue
		}

		// update collector
		collector.reload(host, job)
	}
}
