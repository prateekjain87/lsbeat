package beater

import (
	"fmt"
	"time"
	"io/ioutil"
	"path/filepath"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"

	"github.com/prateekjain87/lsbeat/config"
)

type Lsbeat struct {
    done   chan struct{}
    config config.Config
    client publisher.Client
    lastIndexTime time.Time
}

// Creates beater
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
    config := config.DefaultConfig
    if err := cfg.Unpack(&config); err != nil {
        return nil, fmt.Errorf("Error reading config file: %v", err)
    }
    ls := &Lsbeat{
        done:   make(chan struct{}),
        config: config,
    }
    return ls, nil
}

func (bt *Lsbeat) listDir(dirFile string, beatname string) {
    files, _ := ioutil.ReadDir(dirFile)
    for _, f := range files {
        t := f.ModTime()
        path := filepath.Join(dirFile, f.Name())
        if t.After(bt.lastIndexTime) {
            event := common.MapStr{
                "@timestamp": common.Time(time.Now()),
                "type":       beatname,
                "modtime":    common.Time(t),
                "filename":   f.Name(),
                "path":       path,
                "directory":  f.IsDir(),
                "filesize":   f.Size(),
            }
            bt.client.PublishEvent(event)
        }
        if f.IsDir() {
            bt.listDir(path, beatname)
        }
    }
}


func (bt *Lsbeat) Run(b *beat.Beat) error {
    logp.Info("lsbeat is running! Hit CTRL-C to stop it.")
    bt.client = b.Publisher.Connect()
    ticker := time.NewTicker(bt.config.Period)
    for {
        now := time.Now()
        bt.listDir(bt.config.Path, b.Name) // call listDir
        bt.lastIndexTime = now             // mark Timestamp
        logp.Info("Event sent")
        select {
        case <-bt.done:
            return nil
        case <-ticker.C:
        }
    }
}

func (bt *Lsbeat) Stop() {
	bt.client.Close()
	close(bt.done)
}
