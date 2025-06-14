package network

import (
	"dfinger/common"
	"fmt"
	"net"
	"sync"
	"time"
)

func CheckAlive(check []common.UrlInfo) []common.UrlInfo {
	var wg sync.WaitGroup
	mutex := &sync.Mutex{} // 用来同步访问共享资源

	// 创建一个新的切片，用于存活的 UrlInfo，提前分配空间避免内存拷贝
	aliveInput := make([]common.UrlInfo, 0, len(check))

	// 创建一个任务通道，缓冲大小为线程数
	taskChan := make(chan common.UrlInfo, common.Infos.Threads)

	// 消费者：处理任务的 worker
	for i := 0; i < common.Infos.Threads; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for task := range taskChan {

				address := fmt.Sprintf("%v:%v", task.Host, task.Port)
				conn, err := net.DialTimeout("tcp", address, time.Duration(common.Infos.Timeout)*time.Second)
				//conn, err := net.DialTimeout("tcp", address, time.Duration(common.GlobalContext.Public.Timeout)*time.Millisecond)
				if err == nil {
					// 如果连接成功，表示该端口存活
					mutex.Lock()
					aliveInput = append(aliveInput, task)
					mutex.Unlock()
					conn.Close() // 关闭连接
				}
			}
		}()
	}

	// 生产者：将任务放入通道
	go func() {
		for _, urlInfo := range check {
			taskChan <- urlInfo
		}
		close(taskChan) // 生产者结束后关闭任务通道
	}()

	// 等待所有任务完成
	wg.Wait()

	// 更新全局存活列表
	return aliveInput
}
