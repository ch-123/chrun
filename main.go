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

var lasttime int64 //exec上次运行时间
var pid int32      //exec进程pid

func main() {
	go Exec("go", "run", "main.go")
	watch()
}

//运行命令
func Exec(name string, args ...string) error {
	if t := helper.Time() - lasttime; t < 500 {
		return nil
	}
	lasttime = helper.Time()

	fmt.Println("========== start ==========")
	pid++
	mpid := pid
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
			if pid > mpid {
				exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(cmd.Process.Pid)).Run()
				break
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

//监听文件变化
func watch() {
	//创建一个监控对象
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println(err)
		return
	}
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
			//新加目录 添加监听
			if ev.Op.String() == "CREATE" && !strings.Contains(ev.Name, ".") {
				watch.Add("./" + ev.Name + "/")
			}
			go Exec("go", "run", "main.go")

		case err := <-watch.Errors:
			fmt.Println("chrun error:", err)
		}

	}

}
