// +build linux

package main

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// go run main.go run {cmd} {args}
func main(){
	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		panic("help")
	}
}


func run(){
	fmt.Printf("Running %v \n", os.Args[2:])

	// /proc/self/exeで自分自身を実行できる つまり go run main.go child <cmd> <args> を実行するということ
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// コマンドを実行するときに以下のフラグを渡す
	cmd.SysProcAttr = &unix.SysProcAttr{
		// UTS = ホスト名とNISドメイン名
		// PID = プロセスID
		// NS = マウントポイント
		// https://linuxjm.osdn.jp/html/LDP_man-pages/man7/namespaces.7.html
		Cloneflags: unix.CLONE_NEWUTS | unix.CLONE_NEWPID | unix.CLONE_NEWNS,
		Unshareflags: unix.CLONE_NEWNS,
	}

	must(cmd.Run())
}


func child() {
	fmt.Printf("Running child %v \n", os.Args[2:])

	cg()

	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// hostnameを設定 ユーザー名@ホスト名 ってやつ
	must(unix.Sethostname([]byte("container-masi")))
	// 子プロセスのルートを / に指定
	must(unix.Chroot("/"))
	// ワーキングディレクトリを / に
	must(os.Chdir("/"))

	//
	must(unix.Mount("proc", "proc", "proc", 0, ""))
	must(cmd.Run())

	must(unix.Unmount("proc",0))
}

func cg(){
	cgroups := "/sys/fs/cgroup/"
	pids := filepath.Join(cgroups, "pids")
	os.Mkdir(filepath.Join(pids, "masi"), 0755)

	// cgroupの設定をしていく https://access.redhat.com/documentation/ja-jp/red_hat_enterprise_linux/6/html/resource_management_guide/sec-common_tunable_parameters
	// pids.max は許可するプロセス数
	must(os.WriteFile(filepath.Join(pids, "masi/pids.max"),[]byte("20"),0700))
	// notify_on_release cgroupにタスクがなくなったときにカーネルがrelease_agentファイルの内容を実行するらしい
	must(os.WriteFile(filepath.Join(pids, "masi/notify_on_release"),[]byte("1"),0700))
	// cgroups.procs cgroupで実行中のスレッドグループの一覧が書かれている． cgroupsのtasksファイルに書き込むと，そのスレッドグループはcgroupに移動する
	must(os.WriteFile(filepath.Join(pids, "masi/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())),0700))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
