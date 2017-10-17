package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
)

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

func (sa *selpg_args) process_args() {
	var (
		start_page  int
		end_page    int
		in_filename string
		page_len    int
		form_feed   bool
		print_dest  string
	)
	flag.IntVar(&start_page, "s", 0, "specify `start_page` (default 0)")
	flag.IntVar(&end_page, "e", 0, "specify `end_page` (default 0)")
	flag.IntVar(&page_len, "l", 72, "specify `page_len` (default 72)")
	flag.BoolVar(&form_feed, "f", false, "enable `form_feed` (default unable)")
	flag.StringVar(&print_dest, "d", "", "specify `print_dest` (default )")
	flag.Parse()

	sa.start_page = start_page
	sa.end_page = end_page
	sa.page_len = page_len
	sa.print_dest = print_dest

	if form_feed == false {
		sa.page_type = 'l'
	} else {
		sa.page_type = 'f'
	}

	if flag.NArg() == 0 {
		in_filename = ""
	} else {
		in_filename = flag.Arg(0)

	}
	sa.in_filename = in_filename

}

func (sa *selpg_args) process_input() {
	var (
		fin   *os.File
		err   error
		input *bufio.Reader
		pipe  io.WriteCloser
	)
	/*error when start_page is less than 1*/
	if sa.start_page < 1 {
		fmt.Fprintf(os.Stderr, "%s: start_page %d invalid\n", sa.progname, sa.start_page)
		usage()
		os.Exit(1)
	}
	/*error when end_page is less than 1*/
	if sa.end_page < 1 {
		fmt.Fprintf(os.Stderr, "%s: end_page %d invalid\n", sa.progname, sa.end_page)
		usage()
		os.Exit(2)
	}
	/*error when start_page is greater than end_page*/
	if sa.start_page > sa.end_page {
		fmt.Fprintf(os.Stderr, "%s: start_page %d is greater than end_page %d\n", sa.progname, sa.start_page, sa.end_page)
		os.Exit(3)
	}
	if sa.page_len < 1 {
		fmt.Fprintf(os.Stderr, "%s: page_len %d is less than 1", sa.page_len)
		os.Exit(4)
	}
	/*if filename is not provided, read input from stdin*/
	if sa.in_filename == "" {
		fin = os.Stdin
	} else {
		fin, err = os.Open(sa.in_filename)
		defer fin.Close() /*close the file before the end of this func*/
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(4)
		}
	}
	input = bufio.NewReader(fin)

	if sa.print_dest != "" {
		cmd := exec.Command("lp", "-d", sa.print_dest)
		pipe, err = cmd.StdinPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(5)
		}
		go func() {
			sa.printer(input, pipe)
			defer func() { /*close the pipe before the end of the func*/
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

}

func (sa *selpg_args) printer(input *bufio.Reader, writer io.Writer) {
	var (
		line_ctr int
		page_ctr int
		line     string
		page     string
		err      error
	)
	if sa.page_type == 'l' { /*print specified pages split with specified number of lines*/
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
	} else { /*print specified pages split with '\f'*/
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

func usage() {
	fmt.Printf("\nUSAGE: %s -s start_page -e end_page [ -f | -l lines_per_page ]"+" [ -d dest ] [ in_filename ]\n", os.Args[0])
}

func main() {
	sa := new(selpg_args)
	sa.progname = os.Args[0]
	sa.process_args()
	sa.process_input()
	fmt.Printf("%s: done\n", sa.progname)

}
