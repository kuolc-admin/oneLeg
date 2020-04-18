package scheduler

import (
	"sync"

	"github.com/kawasin73/htask/cron"
)

type Job struct {
	Cancel func()
}

var tasks map[string]*Job
var c *cron.Cron

func init() {
	tasks = map[string]*Job{}

	var wg sync.WaitGroup
	c = cron.NewCron(&wg, cron.Option{
		Workers: 1,
	})
}

func Set(key string, maker func(c *cron.Cron) *Job) {
	if old, ok := tasks[key]; ok {
		old.Cancel()
	}
	tasks[key] = maker(c)
}

func Cancel(key string) {
	if job, ok := tasks[key]; ok {
		job.Cancel()
	}
}
