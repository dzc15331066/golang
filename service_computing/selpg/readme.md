# CLI命令行实用程序开发基础

## 开发平台与语言版本

OS:	Linux amd64

go version: 	1.9

其它工具：cups-pdf，一个支持lp命令的虚拟打印机， [安装和使用指南](http://terokarvinen.com/2011/print-pdf-from-command-line-cups-pdf-lpr-p-pdf)

## 开发任务

使用golang 开发 [开发Linux命令行实用程序](https://www.ibm.com/developerworks/cn/linux/shell/clutil/index.html)中的selpg

## 用法

```shell
 ./selpg -s start_page -e end_page [ -f | -l lines_per_page ] [ -d dest ] [ in_filename ]
```



## 测试

1.以tmp文件作为输入，文件内容为

```
1
2
3
4
5
6
7
8
9
10
11
12
13
14
15
16
17
18
19
20
```



```shell
测试1 	./selpg -s 1 -e 2 -l 5 tmp
```

![test1](/home/dengzhc/图片/test1.png)

```shell
测试2 ./selpg -s 2 -e 3 -l 5 
```

<img src="/home/dengzhc/图片/test2.png">

这是测试从标准输入到标准输出的功能，命令中设置每5行为一页，打印2-3页到标准输出，所以6-15行被打印出来

```
测试3	./selpg -s 2 -e 3 -l 5 -d PDF tmp
```

安装cups-pdf作为一个虚拟打印机，设备地址为PDF，每页5行，打印机打印2-3页

打印结果如下：

<img src="/home/dengzhc/图片/test3.png">

```shell
测试3：	./selpg -s 1 -e 2 -f -d PDF tmp1
```

首先创建带换页符号的文件tmp1

![test4](/home/dengzhc/图片/test4.png)

测试结果如下:

第一页

<img src="/home/dengzhc/图片/test4_1.png">

------

第二页

<img src="/home/dengzhc/图片/test4_2.png">

## 程序结构

### selpg_args



首先是定义打印参数的结构体selpg_args

```go
type selpg_args struct {
	progname    string
	start_page  int
	end_page    int
	in_filename string
	page_len    int  /* default value, can be overridden by "-l number" on command line */
	page_type   byte /* 'l' for lines-delimited, 'f' for form-feed-delimited  */
	/* default is 'l'  */
	print_dest string
}

```

###selpg_args的操作

然后定义selpg_args的操作

```go
func (sa *selpg_args) process_args() {...}/*获取输入参数*/
```

```go
func (sa *selpg_args) process_input() {...}/*检查输入参数的合法性，调用printer打印*/
```

```go
func (sa *selpg_args) printer(input *bufio.Reader, writer io.Writer) {...}/*实现打印功能*/
```

#### process_args

*process_args*解析输入参数时用到了flag包

```go
flag.IntVar(&start_page, "s", 0, "specify `start_page` (default 0)")
	flag.IntVar(&end_page, "e", 0, "specify `end_page` (default 0)")
	flag.IntVar(&page_len, "l", 72, "specify `page_len` (default 72)")
	flag.BoolVar(&form_feed, "f", false, "enable `form_feed` (default unable)")
	flag.StringVar(&print_dest, "d", "", "specify `print_dest` (default )")
	flag.Parse()
```

因为输入文件名没有对应的标签

所以用flag.NArg()和flag.Arg(i)来获取文件名

```go
	if flag.NArg() == 0 {
		in_filename = ""
	} else {
		in_filename = flag.Arg(0)

	}
	sa.in_filename = in_filename
```

####process_input

*process_input*实现打印机打印功能是通过调用` lp -d destination  `作为子进程，获取父进程的输入(来自文件或标准输入)

调用cmd命令用到了`os/exec`包的`exec.Command`

```go
if sa.print_dest != "" {
		cmd := exec.Command("lp", "-d", sa.print_dest)
		pipe, err = cmd.StdinPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(5)
		}
		go func() {
			sa.printer(input, pipe)
			defer func() {
				err = pipe.Close()
				if err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)

				}
			}()
		}()

		_, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(6)
		}

	} else {
		sa.printer(input, os.Stdout)

	}
```

如果不指定打印机，则打印到标准输出。

#### printer

处理两种打印方式: 1.按特定行数(即`-l `的值)为一页打印 2.按`\f`符号分页打印

```go
func (sa *selpg_args) printer(input *bufio.Reader, writer io.Writer) {
	var (
		line_ctr int
		page_ctr int
		line     string
		page     string
		err      error
	)
	if sa.page_type == 'l' {/*print specified pages split with specified lines*/

		line_ctr = 0
		page_ctr = 1
		for {
			line, err = input.ReadString('\n')
			if err != nil && err != io.EOF {
				fmt.Fprintf(os.Stderr, "err:%v\n", err)
				os.Exit(7)
			}
			line_ctr++
			if line_ctr > sa.page_len {
				page_ctr++
				line_ctr = 1
			}
			if page_ctr >= sa.start_page && page_ctr <= sa.end_page {
				io.WriteString(writer, line)
			}
			if err != nil {
				break
			}

		}
	} else {/*print specified pages split with '\f'*/
		page_ctr = 0
		for {
			page, err = input.ReadString('\f')
			if err != nil && err != io.EOF {
				fmt.Fprintf(os.Stderr, "err:%v\n", err)
				os.Exit(7)
			}
			page_ctr++
			if page_ctr >= sa.start_page && page_ctr <= sa.end_page {
				io.WriteString(writer, page)
			}
			if err != nil {

				break
			}

		}
	}

	if page_ctr < sa.start_page {
		fmt.Fprintf(os.Stderr, "%s: start_page (%d) greater than total pages (%d),"+
			"no output written \n", sa.progname, sa.start_page, page_ctr)
	} else if page_ctr < sa.end_page {
		fmt.Fprintf(os.Stderr, "%s: end_page (%d) greater than total pages (%d),"+
			"less ouput than expected \n", sa.progname, sa.end_page, page_ctr)
	}
}

```



###main

然后是`main`函数

```go
func main() {
	sa := new(selpg_args)
	sa.progname = os.Args[0]
	sa.process_args()
	sa.process_input()
	fmt.Printf("%s: done\n", sa.progname)

}
```

这样就实现了一个简单的选页打印程序

