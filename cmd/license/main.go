package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"res-downloader/core"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) == 1 {
		issueInteractive()
		return
	}
	days := flag.Int("days", 0, "有效天数，0 表示长期有效")
	machine := flag.String("machine", "", "电脑识别码，在软件的「软件授权」页复制")
	show := flag.Bool("show-machine", false, "打印这台电脑的识别码")
	flag.Parse()
	if *show {
		fmt.Println(core.MachineCode())
		return
	}
	if *machine == "" {
		fmt.Fprintln(os.Stderr, "请加上 -machine，识别码在软件「系统设置 → 软件授权」里。")
		os.Exit(2)
	}
	if err := printCode(*days, *machine); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func issueInteractive() {
	in := bufio.NewReader(os.Stdin)
	fmt.Println("观澜下载器 · 签发授权码")
	fmt.Print("电脑识别码: ")
	machine, _ := in.ReadString('\n')
	fmt.Print("有效天数，直接回车表示长期有效: ")
	daysText, _ := in.ReadString('\n')
	daysText = strings.TrimSpace(daysText)
	days := 0
	if daysText != "" {
		n, err := strconv.Atoi(daysText)
		if err != nil || n < 0 {
			fmt.Println("天数不正确")
			wait(in)
			os.Exit(1)
		}
		days = n
	}
	if err := printCode(days, strings.TrimSpace(machine)); err != nil {
		fmt.Println(err.Error())
		wait(in)
		os.Exit(1)
	}
	wait(in)
}

func printCode(days int, machine string) error {
	code, err := core.IssueLicense(days, machine)
	if err != nil {
		return err
	}
	fmt.Println(code)
	return nil
}

func wait(in *bufio.Reader) {
	fmt.Print("按回车关闭")
	_, _ = in.ReadString('\n')
}
