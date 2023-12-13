package arc_volume

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kiga-hub/arc-storage/pkg/util"

	"github.com/kiga-hub/arc/logging"
)

// Tasker - 任务接口
type Tasker interface {
	getCtx() context.Context
	waitStart() chan struct{}
	start()
	waitComplete() chan struct{}
	complete()
	setExpire()
	isExpire() bool
	getQueueIDSourceKey() string
	getTaskID() int64
	getTaskType() string

	timeout()
	handle()
}

// 任务
type task struct {
	queueIDSourceKey string
	taskID           int64
	ctx              context.Context // 超时控制
	startCh          chan struct{}   // 用于等待异步任务开始执行，通过此通道变量控制任务是否开始执行
	completeCh       chan struct{}   // 用于等待异步任务执行，通过此通道变量控制任务是否执行完毕
	expire           atomic.Value    // 设置任务失效，已经进入队列的任务，在失效后会跳过执行
	taskType         string          // 任务类型
}

func newTask(ctx context.Context, queueIDSourceKey string, taskType string) *task {
	t := &task{
		queueIDSourceKey: queueIDSourceKey,
		taskID:           time.Now().UnixNano(),
		ctx:              ctx,
		startCh:          make(chan struct{}),
		completeCh:       make(chan struct{}),
		expire:           atomic.Value{},
		taskType:         taskType,
	}
	return t
}
func (t *task) getCtx() context.Context {
	return t.ctx
}
func (t *task) waitStart() chan struct{} {
	return t.startCh
}
func (t *task) start() {
	close(t.startCh)
}
func (t *task) waitComplete() chan struct{} {
	return t.completeCh
}
func (t *task) complete() {
	close(t.completeCh)
}
func (t *task) setExpire() {
	t.expire.Store(1)
}
func (t *task) isExpire() bool {
	d := t.expire.Load()
	return d != nil
}
func (t *task) getQueueIDSourceKey() string {
	return t.queueIDSourceKey
}
func (t *task) getTaskID() int64 {
	return t.taskID
}
func (t *task) getTaskType() string {
	return t.taskType
}

// 传感器队列
type queue struct {
	logger  logging.ILogger
	queueID int
	taskCh  chan Tasker
	once    sync.Once
}

func newQueue(logger logging.ILogger, queueID int, l int) *queue {
	q := &queue{
		logger:  logger,
		queueID: queueID,
		taskCh:  make(chan Tasker, l),
	}

	go q.start()

	return q
}

func (q *queue) start() {

	for task := range q.taskCh {

		if task.isExpire() {
			q.logger.Infow("rwqueue skip timeout task", "queue_id", q.queueID, "task_id", task.getTaskID())
			continue
		}

		task.start() //状态变更开始执行

		go func(t Tasker) {

			defer func() {
				t.complete() //状态变更执行完毕
			}()

			t.handle()

		}(task)

		select {
		case <-task.getCtx().Done():
			break
		case <-task.waitComplete():
			break
		}

	}

	q.logger.Infow("rwqueue queue closed", "queue_id", q.queueID)
}

func (q *queue) close() {
	q.once.Do(func() {
		close(q.taskCh)
	})
}

// Queue - 队列封装实体
type Queue struct {
	logger    logging.ILogger
	queueList []*queue
	queueLen  int
	queueNum  int
}

// NewQueue -
func NewQueue(logger logging.ILogger, l int, n int) *Queue {

	rwq := &Queue{
		logger:    logger,
		queueList: make([]*queue, n),
		queueLen:  l,
		queueNum:  n,
	}

	for i := 0; i < rwq.queueNum; i++ {
		rwq.queueList[i] = newQueue(rwq.logger, i, rwq.queueLen) //创建队列
	}

	go rwq.monitor()

	return rwq
}

func (rwq *Queue) getQueue(queueIDSourceKey string) *queue {
	value := util.BytesToUint64([]byte(queueIDSourceKey))
	hashKey := uint64(rwq.queueNum - 1)
	i := value & hashKey
	return rwq.queueList[i]
}

func (rwq *Queue) monitor() {
	for {
		time.Sleep(time.Second)
		for i, q := range rwq.queueList {
			setQueueLenMetric(i, float64(len(q.taskCh)))
		}
	}
}

// Close -
func (rwq *Queue) Close() {
	var wg sync.WaitGroup
	for _, q := range rwq.queueList {
		wg.Add(1)
		go func(queue *queue) {
			defer wg.Done()
			queue.close()
		}(q)
	}
	wg.Wait()
}

// DoTask -
func (rwq *Queue) DoTask(task Tasker) {

	queue := rwq.getQueue(task.getQueueIDSourceKey())

	// 等待写入队列
	select {
	case <-task.getCtx().Done():
		task.timeout()

		rwq.logger.Warnw("rwqueue wait insert timeout", "queue_id_source_key", task.getQueueIDSourceKey(), "task_id", task.getTaskID())
		addTaskTimeoutTimesMetric(task.getTaskType())

		return
	case queue.taskCh <- task:
		addQueueInputNumMetric(queue.queueID)
		break
	}

	// 进入队列后等待开始执行
	select {
	case <-task.getCtx().Done():
		task.setExpire()
		task.timeout()

		rwq.logger.Warnw("rwqueue wait start timeout", "queue_id_source_key", task.getQueueIDSourceKey(), "task_id", task.getTaskID())
		addTaskTimeoutTimesMetric(task.getTaskType())

		return
	case <-task.waitStart():
		break
	}

	// 开始执行后等待执行结果
	select {
	case <-task.getCtx().Done():
		task.timeout()

		rwq.logger.Warnw("rwqueue wait complete timeout", "queue_id_source_key", task.getQueueIDSourceKey(), "task_id", task.getTaskID())
		addTaskTimeoutTimesMetric(task.getTaskType())

		return
	case <-task.waitComplete():
		break
	}

}
