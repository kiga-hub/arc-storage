package crontab

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

var (
	username     = "root"
	password     = "taosdata"
	cfgpath      = "/etc/taos"
	databasename = "arc"
	taosdump     = "/usr/bin/taosdump"
)

// DumpParams TDEngine dump command params
type DumpParams struct {
	TaosHost        string // tdengine docker host
	TaosPort        string // tdengine doceker port
	TaosBackUpStart string // 开始异步备份数据时间
	TaosBackUpStop  string // 停止异步备份数据时间
	SyncBackUpStart string // 开始同步文件备份数据时间
	SyncBackUpStop  string // 停止同步文件备份数据时间
	Duration        string // 异步备份数据库间隔(12h,24h)
	Path            string // 磁盘挂载路径
	Tablebatch      string // 一个文件导入的表的个数，可控制输出文件大小，默认为10，可选项
	Tablename       string // 指定导出数据库数据表的名，内部测试，可选项
	Threadnum       string // 异步导出数据库启动的线程数，默认为5，可选项
	Databatch       string // 指定一条import中包含记录的条数，可提高导入速度，默认为5，可选项
	Exporttype      string // 导出类型，0,1,2 指定是否指定数据库，数据表
}

// Crontab 定时任务
type Crontab struct {
	Stopchan           chan struct{}
	lk                 sync.Mutex
	MigrationFunc      func(DumpParams) error
	Params             DumpParams
	SyncBackupPath     string
	IsSyncMigrateData  bool // 开始进行同步数据备份标志位
	IsASyncMigrateData bool //开始进行异步数据备份标志位
}

// Start - start crontab tasks.
func (c *Crontab) Start() error {
	c.lk.Lock()
	defer c.lk.Unlock()

	if c.Stopchan != nil {
		return fmt.Errorf("Crontab already started")
	}
	c.Stopchan = make(chan struct{})
	c.MigrationFunc = exportDataBaseFunc
	c.IsSyncMigrateData = false
	c.IsASyncMigrateData = false

	return nil
}

// Stop - stop crontab tasks.
func (c *Crontab) Stop() error {
	c.lk.Lock()
	defer c.lk.Unlock()

	if c.Stopchan == nil {
		return fmt.Errorf("stopchan errors")
	}
	close(c.Stopchan)
	return nil
}

// IsBackups check is bakcups or not.
func (c *Crontab) IsBackups() bool {
	c.lk.Lock()
	defer c.lk.Unlock()

	return c.IsSyncMigrateData || c.IsASyncMigrateData
}

// StartBackups Reset params
func (c *Crontab) StartBackups(issyncmigratedata bool, syncbackuppath string, params DumpParams) error {
	c.lk.Lock()
	defer c.lk.Unlock()

	c.IsSyncMigrateData = issyncmigratedata
	c.Params = params
	c.SyncBackupPath = syncbackuppath

	if c.Stopchan == nil {
		return fmt.Errorf("Stop chan errors")
	}
	if c.IsSyncMigrateData {
		c.CrontabHandler()
	}
	return nil
}

// CrontabHandler CrontabHandler
func (c *Crontab) CrontabHandler() {
	go func() {
		for {
			select {
			case <-c.Stopchan:
				return
			default:
				duration, err := strconv.Atoi(c.Params.Duration)
				if err != nil {
					duration = 24
				}
				now := time.Now().UTC()
				if c.Params.SyncBackUpStart == "" {
					c.Params.SyncBackUpStart = now.Format("2006-01-02 15:04:05.999")
				}
				// Count next zero time
				next := now.UTC().Add(time.Hour * time.Duration(duration)) //24
				// next := now.Add(time.Second * 1)
				next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
				t := time.NewTimer(next.Sub(now))
				<-t.C
				c.Params.SyncBackUpStart = next.Format("2006-01-02 15:04:05.999")
				err = c.MigrationFunc(c.Params)
				if err != nil {
					continue
				}
			}
		}
	}()
}

// exportDataBaseFunc Processing parameters
func exportDataBaseFunc(dumparams DumpParams) error {
	starttime := dumparams.TaosBackUpStart
	stoptime := dumparams.TaosBackUpStop

	s, err := time.Parse("2006-01-02 15:04:05.999", starttime)
	if err != nil {
		return fmt.Errorf("time.Parse:  %s", err)
	}
	e, err := time.Parse("2006-01-02 15:04:05.999", stoptime)
	if err != nil {
		return fmt.Errorf("time.Parse:  %s", err)
	}
	starttime = strconv.FormatInt(s.UTC().UnixNano()/1e3, 10)
	stoptime = strconv.FormatInt(e.UTC().UnixNano()/1e3, 10)

	dockercommand := "docker exec " + dumparams.TaosHost + " /bin/bash -c "
	exportdatabasecommand := taosdump + " -h " + dumparams.TaosHost + " -P " + dumparams.TaosPort + " -c " + cfgpath + " -o " + dumparams.Path + " -u " + username + " -p " + password + " -B " + databasename + " -S " + starttime + " -E " + stoptime + " -t " + dumparams.Threadnum + " - T " + dumparams.Tablebatch + " -N " + dumparams.Databatch
	err = TDEngineDataMigration(dockercommand + " \"" + exportdatabasecommand + " \"")
	if err != nil {
		return fmt.Errorf("TDEngineDataMigration:  %s", err)
	}
	return nil
}

// TDEngineDataMigration docker command
func TDEngineDataMigration(command string) error {
	cmd := exec.Command("/bin/bash", "-c", command)
	//创建获取命令输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error:can not obtain stdout pipe for command: %s", err)
	}
	//执行命令
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error:The command is err: %s", err)
	}

	//使用带缓冲的读取器
	outputBuf := bufio.NewReader(stdout)
	for {
		//一次获取一行,_ 获取当前行是否被读完
		_, _, err := outputBuf.ReadLine() //output, _, err
		if err != nil {
			// 判断是否到文件的结尾了否则出错
			if err.Error() != "EOF" {
				return fmt.Errorf("error :%s", err)
			}
			break
		}
		// return fmt.Errorf("%s\n", string(output))
	}
	//wait 方法会一直阻塞到其所属的命令完全运行结束为止
	if err = cmd.Wait(); err != nil {
		return fmt.Errorf("wait: %s", err.Error())
	}
	return nil
}
