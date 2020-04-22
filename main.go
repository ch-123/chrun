package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ch-123/helper"
	"github.com/fsnotify/fsnotify"
)

var lastChangeTime int64                 //上次文件改变时间
var runNum int                           //exec运行编号
var taskkill chan bool = make(chan bool) //关闭上个进程

func main() {
	go Exec("go", "run", "main.go")
	watch()
}

//运行命令
func Exec(name string, args ...string) error {
	fmt.Println("========== start ==========")
	go func() {
		taskkill <- true
	}()
	runNum++
	thisRunNum := runNum
	cmd := exec.Command(name, args...)
	stderr, _ := cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		fmt.Println("exec the cmd ", name, " failed")
		return err
	}
	//每次运行 杀掉上次运行进程
	go func() {
		for {
			<-taskkill
			if runNum > thisRunNum {
				exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(cmd.Process.Pid)).Run()
				return
			}
		}
	}()
	// 正常日志
	logScan := bufio.NewScanner(stdout)
	go func() {
		for logScan.Scan() {
			fmt.Println(logScan.Text())
		}
	}()
	// 错误日志
	scan := bufio.NewScanner(stderr)
	go func() {
		for scan.Scan() {
			s := scan.Text()
			fmt.Println("error: ", s)
		}
	}()
	cmd.Wait()
	return nil
}

//文件发生改变
func fileChange(op, name string, watch *fsnotify.Watcher) {
	//500毫秒内发生多次文件改变  只执行一次
	if t := helper.Time() - lastChangeTime; t < 500 {
		return
	}
	lastChangeTime = helper.Time()
	//新加目录 添加监听
	if op == "CREATE" && !strings.Contains(name, ".") {
		watch.Add("./" + name + "/")
	}
	Exec("go", "run", "main.go")
}

//监听文件变化
func watch() {
	//创建一个监控对象
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer watch.Close()
	//添加要监控的对象，文件或文件夹
	err = watch.Add("./")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.Contains(path, ".") {
			watch.Add(path + "/")
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
	for {
		select {
		case ev, _ := <-watch.Events:
			go fileChange(ev.Op.String(), ev.Name, watch)
		case err := <-watch.Errors:
			fmt.Println("监听文件错误:", err)
		}

	}

}
