package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)
// test1
// 方法1：使用 time.After 实现定时停止
func stopAfterDuration(duration time.Duration) {
	fmt.Printf("程序将在 %v 后自动停止...\n", duration)

	// 模拟程序运行
	go func() {
		for {
			fmt.Println("程序正在运行中...")
			time.Sleep(1 * time.Second)
		}
	}()

	// 定时停止
	<-time.After(duration)
	fmt.Println("时间到！程序自动停止。")
	os.Exit(0)
}

// 方法2：使用 context.WithTimeout 实现定时停止
func stopWithContext(duration time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	fmt.Printf("程序将在 %v 后自动停止...\n", duration)

	// 模拟程序运行
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				fmt.Println("程序正在运行中...")
				time.Sleep(1 * time.Second)
			}
		}
	}()

	// 等待超时
	<-ctx.Done()
	fmt.Println("时间到！程序自动停止。")
}

// 方法3：使用 time.NewTimer 实现定时停止
func stopWithTimer(duration time.Duration) {
	timer := time.NewTimer(duration)

	fmt.Printf("程序将在 %v 后自动停止...\n", duration)

	// 模拟程序运行
	go func() {
		for {
			fmt.Println("程序正在运行中...")
			time.Sleep(1 * time.Second)
		}
	}()

	// 等待定时器触发
	<-timer.C
	fmt.Println("时间到！程序自动停止。")
}

// 监控结构体
type Monitor struct {
	printCount int64 // 使用原子操作确保线程安全
	stopChan   chan bool
	startTime  time.Time // 记录开始时间
}

// 新建监控器
func NewMonitor() *Monitor {
	return &Monitor{
		printCount: 0,
		stopChan:   make(chan bool),
		startTime:  time.Now(), // 记录创建时间
	}
}

// 增加打印计数
func (m *Monitor) IncrementPrintCount() {
	atomic.AddInt64(&m.printCount, 1)
}

// 获取当前打印计数
func (m *Monitor) GetPrintCount() int64 {
	return atomic.LoadInt64(&m.printCount)
}

// 启动监控goroutine
func (m *Monitor) StartMonitoring() {
	go func() {
		ticker := time.NewTicker(1 * time.Second) // 每1秒报告一次统计
		defer ticker.Stop()

		fmt.Println("[监控] 打印次数监控已启动，每1秒报告一次统计信息")

		for {
			select {
			case <-ticker.C:
				count := m.GetPrintCount()
				// 计算程序运行的总时间（秒）
				elapsedSeconds := time.Since(m.startTime).Seconds()
				var avgPerSecond float64
				if elapsedSeconds > 0 {
					avgPerSecond = float64(count) / elapsedSeconds
				}
				fmt.Printf("[监控] 当前总打印次数: %d, 运行时间: %.1f秒, 平均每秒: %.2f次\n",
					count, elapsedSeconds, avgPerSecond)
			case <-m.stopChan:
				finalCount := m.GetPrintCount()
				totalSeconds := time.Since(m.startTime).Seconds()
				var finalAvg float64
				if totalSeconds > 0 {
					finalAvg = float64(finalCount) / totalSeconds
				}
				fmt.Printf("[监控] 监控已停止，最终打印次数: %d, 总运行时间: %.1f秒, 最终平均每秒: %.2f次\n",
					finalCount, totalSeconds, finalAvg)
				return
			}
		}
	}()
}

// 停止监控
func (m *Monitor) StopMonitoring() {
	close(m.stopChan)
}

// 方法4：支持手动停止和定时停止的组合（添加监控功能）
func stopWithGracefulShutdown(duration time.Duration) {
	stopWithGracefulShutdownConcurrent(duration, 4) // 默认4个并发
}

// 方法4-增强版：支持自定义并发数量的优雅关闭
func stopWithGracefulShutdownConcurrent(duration time.Duration, concurrency int) {
	// 创建监控器
	monitor := NewMonitor()

	// 启动监控goroutine
	monitor.StartMonitoring()

	// 创建信号通道
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 创建定时器
	timer := time.NewTimer(duration)

	fmt.Printf("程序将在 %v 后自动停止，或按 Ctrl+C 手动停止...\n", duration)
	fmt.Printf("同时启动了打印次数监控功能，并发数量: %d\n", concurrency)

	// 模拟程序运行 - 启动指定数量的并发goroutine
	done := make(chan bool)

	// 方案1：多个并发goroutine，每个每秒输出1次，但错开启动时间
	if concurrency > 1 {
		interval := 1000 / concurrency // 计算启动间隔（毫秒）
		for i := 0; i < concurrency; i++ {
			go func(id int) {
				// 错开启动时间，使输出更均匀分布
				time.Sleep(time.Duration(id*interval) * time.Millisecond)

				for {
					select {
					case <-done:
						return
					default:
						fmt.Printf("💼 工作线程-%d: 程序正在运行中...\n", id+1)
						monitor.IncrementPrintCount() // 增加打印计数
						time.Sleep(1 * time.Second)
					}
				}
			}(i)
		}
	} else {
		// 方案2：单个goroutine高频输出
		go func() {
			interval := 1000 / concurrency // 计算输出间隔（毫秒）
			for {
				select {
				case <-done:
					return
				default:
					fmt.Println("⚡ 高频输出: 程序正在运行中...")
					monitor.IncrementPrintCount() // 增加打印计数
					time.Sleep(time.Duration(interval) * time.Millisecond)
				}
			}
		}()
	}

	// 等待信号或定时器
	select {
	case <-sigChan:
		fmt.Println("\n收到停止信号，程序正在优雅关闭...")
		timer.Stop()
	case <-timer.C:
		fmt.Println("时间到！程序自动停止。")
	}

	// 停止监控
	monitor.StopMonitoring()

	close(done)
	time.Sleep(500 * time.Millisecond) // 给程序一点时间清理
	fmt.Printf("程序已安全停止。总共打印了 %d 次消息。\n", monitor.GetPrintCount())
}

func main() {
	// 设置停止时间（例如：12秒后停止）
	duration := 12 * time.Second

	fmt.Println("=== 定时停止程序示例（带并发监控）===")
	fmt.Println("🚀 并发输出优化版本")
	fmt.Println("请选择一种实现方式：")
	fmt.Println("1. 使用 time.After")
	fmt.Println("2. 使用 context.WithTimeout")
	fmt.Println("3. 使用 time.NewTimer")
	fmt.Println("4. 支持手动和定时停止（带并发监控）")
	fmt.Println()

	// 演示不同的并发配置
	fmt.Println("🎯 当前配置：每秒4个并发输出")
	stopWithGracefulShutdown(duration) // 默认4个并发

	// 取消注释以下行来测试不同的并发数量：
	// fmt.Println("🎯 测试：每秒8个并发输出")
	// stopWithGracefulShutdownConcurrent(duration, 8)

	// fmt.Println("🎯 测试：每秒2个并发输出")
	// stopWithGracefulShutdownConcurrent(duration, 2)

	// fmt.Println("🎯 测试：每秒10个高频输出")
	// stopWithGracefulShutdownConcurrent(duration, 10)

	// 取消注释以下行来测试其他方法：
	// stopAfterDuration(duration)
	// stopWithContext(duration)
	// stopWithTimer(duration)
}
// test
// test
// test
