package cache

import (
	"github.com/sirupsen/logrus"
	"sort"
	"sync"
	"time"
)

var Cache *ExpiredMap

// 1 Stopped, 2 Running, 3 Pending
const (
	delChannelCap = 100
	Stopped       = iota
	Running
	Pending
)

var StateMap = map[int]string{
	Stopped: "Stopped",
	Running: "Running",
	Pending: "Pending",
}

type Val struct {
	Data           []*ExecStatus `json:"data,omitempty"`
	State          int           `json:"state"`
	ExpiredTimes   int64         `json:"expired_times"`
	StartTimes     int64         `json:"start_times"`
	CompletedTimes int64         `json:"completed_times"`
}

type ExecStatus struct {
	Name     string `json:"name"`
	Step     int    `json:"step"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
}

type ExpiredMap struct {
	m       map[string]*Val
	timeMap map[int64][]string
	stop    chan struct{}
	lck     *sync.Mutex
}

func NewExpiredMap() *ExpiredMap {
	e := &ExpiredMap{
		m:       make(map[string]*Val),
		timeMap: make(map[int64][]string),
		stop:    make(chan struct{}),
		lck:     new(sync.Mutex),
	}
	// 定期清理数据. 60秒
	go e.run(time.Now().Unix())
	return e
}

type delMsg struct {
	keys []string
	t    int64
}

// background goroutine Actively delete expired keys
//the actual deletion time of the data is a little later than
//the time it should be deleted, and this error will be resolved during the query.
func (e *ExpiredMap) run(now int64) {
	t := time.NewTicker(time.Second * 1)
	defer t.Stop()
	delCh := make(chan *delMsg, delChannelCap)
	go func() {
		for v := range delCh {
			logrus.Info("clean up expire tasks ", v.keys)
			e.multiDelete(v.keys, v.t)
		}
	}()
	for {
		select {
		case <-t.C:
			// Using the form of now++ here, using time.Now().Unix() directly may
			//cause the time to skip 1s, resulting in the key not being deleted.
			now++
			e.lck.Lock()
			if keys, found := e.timeMap[now]; found {
				e.lck.Unlock()
				delCh <- &delMsg{keys: keys, t: now}
			} else {
				e.lck.Unlock()
			}
		case <-e.stop:
			close(delCh)
			return
		}
	}
}

type ListData struct {
	ID             string `json:"id"`
	State          int    `json:"state"`
	ExpiredTimes   int64  `json:"expired_times"`
	StartTimes     int64  `json:"start_times"`
	CompletedTimes int64  `json:"completed_times"`
}

type ListByStartTimes []*ListData

func (l ListByStartTimes) Len() int           { return len(l) }
func (l ListByStartTimes) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l ListByStartTimes) Less(i, j int) bool { return l[i].StartTimes < l[j].StartTimes }

func (e *ExpiredMap) GetAllByStartTimes() (res ListByStartTimes) {
	e.lck.Lock()
	for k, v := range e.m {
		res = append(res, &ListData{
			ID:             k,
			State:          v.State,
			ExpiredTimes:   v.ExpiredTimes,
			StartTimes:     v.StartTimes,
			CompletedTimes: v.CompletedTimes,
		})
	}
	e.lck.Unlock()
	// sort by StartTimes
	sort.Sort(res)
	return
}

type ListByCompletedTimes []*ListData

func (l ListByCompletedTimes) Len() int           { return len(l) }
func (l ListByCompletedTimes) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l ListByCompletedTimes) Less(i, j int) bool { return l[i].CompletedTimes < l[j].CompletedTimes }

func (e *ExpiredMap) GetAllByEndTimes() (res ListByCompletedTimes) {
	e.lck.Lock()
	for k, v := range e.m {
		res = append(res, &ListData{
			ID:             k,
			State:          v.State,
			ExpiredTimes:   v.ExpiredTimes,
			StartTimes:     v.StartTimes,
			CompletedTimes: v.CompletedTimes,
		})
	}
	e.lck.Unlock()
	// sort by CompletedTimes
	sort.Sort(res)
	return
}

type ListByExpiredTimes []*ListData

func (l ListByExpiredTimes) Len() int           { return len(l) }
func (l ListByExpiredTimes) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l ListByExpiredTimes) Less(i, j int) bool { return l[i].ExpiredTimes < l[j].ExpiredTimes }

func (e *ExpiredMap) GetAllByExpiredTimes() (res ListByExpiredTimes) {
	e.lck.Lock()
	for k, v := range e.m {
		res = append(res, &ListData{
			ID:             k,
			State:          v.State,
			ExpiredTimes:   v.ExpiredTimes,
			StartTimes:     v.StartTimes,
			CompletedTimes: v.CompletedTimes,
		})
	}
	e.lck.Unlock()
	// sort by ExpiredTimes
	sort.Sort(res)
	return
}

type ExecResults []*ExecStatus

func (e ExecResults) Len() int           { return len(e) }
func (e ExecResults) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e ExecResults) Less(i, j int) bool { return e[i].Step < e[j].Step }

func (e *ExpiredMap) Set(key string, val *Val) {
	e.lck.Lock()
	defer e.lck.Unlock()
	e.m[key] = val
	e.timeMap[val.ExpiredTimes] = append(e.timeMap[val.ExpiredTimes], key) //过期时间作为key，放在map中
}

func (e *ExpiredMap) Get(key string) (found bool, value *Val) {
	e.lck.Lock()
	defer e.lck.Unlock()
	if found = e.checkDeleteKey(key); !found {
		return
	}
	value = e.m[key]
	return
}

func (e *ExpiredMap) Delete(key string) {
	e.lck.Lock()
	delete(e.m, key)
	e.lck.Unlock()
}

func (e *ExpiredMap) Remove(key string) {
	e.Delete(key)
}

func (e *ExpiredMap) multiDelete(keys []string, t int64) {
	e.lck.Lock()
	defer e.lck.Unlock()
	delete(e.timeMap, t)
	for _, key := range keys {
		delete(e.m, key)
	}
}

// TTL Returns the remaining time to live for the key. The key does not exist and returns a negative number
func (e *ExpiredMap) TTL(key string) int64 {
	e.lck.Lock()
	defer e.lck.Unlock()
	if !e.checkDeleteKey(key) {
		return -1
	}
	return e.m[key].ExpiredTimes - time.Now().Unix()
}

func (e *ExpiredMap) Clear() {
	e.lck.Lock()
	defer e.lck.Unlock()
	e.m = make(map[string]*Val)
	e.timeMap = make(map[int64][]string)
}

func (e *ExpiredMap) Close() {
	e.lck.Lock()
	defer e.lck.Unlock()
	e.stop <- struct{}{}
	e.m = nil
	e.timeMap = nil
}

func (e *ExpiredMap) Stop() {
	e.Close()
}

func (e *ExpiredMap) checkDeleteKey(key string) bool {
	if val, found := e.m[key]; found {
		if val.ExpiredTimes <= time.Now().Unix() {
			delete(e.m, key)
			//delete(e.timeMap, val.expiredTime)
			return false
		}
		return true
	}
	return false
}
