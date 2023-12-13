package arc_volume

import (
	"context"
	"net/http"
	"time"

	"github.com/kiga-hub/arc/utils"
)

// 任务种类
const taskTypeRead = "read"
const taskTypeWrite = "write"

// 读取任务 - 该自定义类型需要继承task，并确保实现Tasker接口
type readTask struct {
	*task

	handleObj     *ArcVolumeCache
	paramSensorID string
	paramFileType string
	paramStart    time.Time
	paramEnd      time.Time

	data []byte
	err  *utils.ResponseV2
}

func createReadTask(ctx context.Context, queueIDSourceKey string, bfc *ArcVolumeCache, sensorID, fileType string, t1, t2 time.Time) *readTask {
	return &readTask{
		task: newTask(ctx, queueIDSourceKey, taskTypeRead),

		handleObj:     bfc,
		paramSensorID: sensorID,
		paramFileType: fileType,
		paramStart:    t1,
		paramEnd:      t2,
	}
}

func (t *readTask) handle() {
	s := time.Now()
	t.data, t.err = t.handleObj.readDataLogic(t.ctx, t.paramSensorID, t.paramFileType, t.paramStart, t.paramEnd)
	if t.err != nil && t.err.Code == http.StatusGatewayTimeout { //超时
		addTaskTimeoutTimesMetric(t.getTaskType())
	} else {
		addTaskCostMetric(t.getTaskType(), time.Since(s).Seconds())
	}
}
func (t *readTask) timeout() {
	if t.err == nil {
		t.err = &utils.ResponseV2{
			Code: http.StatusGatewayTimeout,
			Msg:  http.StatusText(http.StatusGatewayTimeout),
		}
	}
}

// 写入任务
type writeTask struct {
	*task

	handleObj *ArcVolumeCache
	paramData *ArcVolume

	err error
}

func createWriteTask(ctx context.Context, queueIDSourceKey string, bfc *ArcVolumeCache, data *ArcVolume) *writeTask {
	return &writeTask{
		task: newTask(ctx, queueIDSourceKey, taskTypeWrite),

		handleObj: bfc,
		paramData: data,
	}
}

func (t *writeTask) handle() {
	s := time.Now()
	defer func() {
		addTaskCostMetric(t.getTaskType(), time.Since(s).Seconds())
	}()

	t.err = t.handleObj.writeDataLogic(t.ctx, t.paramData)
}
func (t *writeTask) timeout() {
	if t.err == nil {
		t.err = http.ErrHandlerTimeout
	}
}
