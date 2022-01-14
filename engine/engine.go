package engine

import (
	"fmt"
	"github.com/Jeffail/tunny"
	"github.com/natessilva/dag"
	"github.com/sirupsen/logrus"
	"github.com/xmapst/osreapi/cache"
	"sort"
	"sync"
	"time"
)

type ExecJobs struct {
	lock      *sync.Mutex
	results   cache.ExecResults
	TaskID    string
	Jobs      []*Jobs
	Val       *cache.Val
	KeyExpire time.Duration
}

type Jobs struct {
	Name           string
	CommandType    string
	CommandContent string
	DependsOn      []string
	EnvVars        []string
	ExecTimeout    time.Duration
}

var Pool *tunny.Pool

func NewExecPool(size int) {
	Pool = tunny.NewFunc(size, worker)
}

func Process(taskID string, jobs []*Jobs, keyExpire time.Duration) {
	// 临时存储
	val := &cache.Val{
		State: cache.Pending,
	}
	// 插入数据
	cache.Cache.Set(taskID, val)
	Pool.Process(ExecJobs{
		lock:      new(sync.Mutex),
		TaskID:    taskID,
		Jobs:      jobs,
		Val:       val,
		KeyExpire: keyExpire,
	})
}

func worker(i interface{}) interface{} {
	e, ok := i.(ExecJobs)
	if !ok {
		logrus.Error("input problem")
		return nil
	}
	// 设置过期时间
	e.KeyExpire += 1 * time.Hour
	e.Val.ExpiredTimes = time.Now().Add(e.KeyExpire).Unix()
	e.Val.StartTimes = time.Now().UnixNano()
	e.Val.State = cache.Running
	// 更新数据
	cache.Cache.Set(e.TaskID, e.Val)
	// 开始编排jobs
	var d dag.Runner
	// create a directed graph, where each vertex in the graph
	// is a pipeline job.
	for k, job := range e.Jobs {
		j := job
		step := k
		d.AddVertex(job.Name, func() error {
			return e.execJob(step, j)
		})
	}
	// create the vertex edges from the values configured in the
	// depends_on attribute.
	for _, job := range e.Jobs {
		for _, dep := range job.DependsOn {
			d.AddEdge(dep, job.Name)
		}
	}
	if err := d.Run(); err != nil {
		e.results = cache.ExecResults{
			{
				Name:     fmt.Sprintf("%s-init", e.TaskID),
				Step:     -1,
				ExitCode: 255,
				Output:   err.Error(),
			},
		}
	}
	// sort by Step
	sort.Sort(e.results)
	e.Val.Data = e.results
	e.Val.State = cache.Stopped
	e.Val.CompletedTimes = time.Now().UnixNano()
	cache.Cache.Set(e.TaskID, e.Val)
	return nil
}

func (e *ExecJobs) execJob(step int, job *Jobs) error {
	if job.EnvVars != nil {
		logrus.Infof("task %s step %d inject external environment variables %v", e.TaskID, step, job.EnvVars)
	}
	logrus.Infof("task %s step %d exec job %s", e.TaskID, step, job.Name)
	var result = &cache.ExecStatus{
		Name:     job.Name,
		Step:     step,
		ExitCode: 255,
	}
	defer func() {
		e.lock.Lock()
		e.results = append(e.results, result)
		e.lock.Unlock()
	}()
	cmd := &Cmd{
		TaskID:          e.TaskID,
		Step:            step,
		Name:            job.Name,
		Shell:           job.CommandType,
		Content:         job.CommandContent,
		ExternalEnvVars: job.EnvVars,
		ExecTimeout:     job.ExecTimeout,
	}
	if err := cmd.Initial(); err != nil {
		result.Output = err.Error()
		return err
	}
	result.ExitCode, result.Output = cmd.ExecScript(e.TaskID)
	if result.ExitCode != 0 {
		logrus.Error(e.TaskID, job.Name, " exec error")
		return fmt.Errorf("task id %s step %d job name %s exec error", e.TaskID, step, job.Name)
	}
	return nil
}
