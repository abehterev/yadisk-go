package main

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"io"
	//"io/ioutil"
	"log"
	"os"
	yadisk "github.com/abehterev/yadisk-go"
)

var (
	oAuthToken = "some_token_for_yandex"
)

type IYaDisk interface {
	ReceiveMainRes() (err error)
	GetCurl(name string) (curlstr string, err error)
	GetData(name string, writer io.Writer) (err error)
	PutData(name string, reader io.Reader) (err error)
	DelRes(path string) (err error)
}

func usage() {
	fmt.Print(
		"Usage:\n",
		os.Args[0]+" <command> <param>\n",
		"\tcommands and params:\n",
		"\t\tinfo\n",
		"\t\tsend <file>\n",
		"\t\tget <file>\n",
		"\t\tcurl <file>\n",
		"\t\tdel <path>\n",
	)
	os.Exit(1)
}

func getInfo(yd IYaDisk) {
	err := yd.ReceiveMainRes()
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	spew.Dump(yd)
}

func delRes(yd IYaDisk, path string) {
	err := yd.DelRes(path)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	fmt.Printf("\nResource %s was deleted\n", path)
}

func getCurl(yd IYaDisk, filename string) {
	curl, err := yd.GetCurl(filename)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	fmt.Println(curl)
}

func getFile(yd IYaDisk, filename string) {

	chunks := int64(0)
	br, bw := io.Pipe() // Body reader-writer
	fr, fw := io.Pipe() // File reader-writer

	go func() {
		//fmt.Println("gofunc Get")
		err := yd.GetData(filename, bw)
		defer bw.Close()

		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	}()

	go func() {
		//fmt.Println("gofunc Read")
		data := make([]byte, 1*1024*1024)
		var total uint64
		for {
			n, err := br.Read(data)
			total += uint64(n)
			if err != nil {
				if err == io.EOF && n == 0 {
					break
				}
				log.Fatal(err)
			}
			/*
				fmt.Print("<.> ")
				fmt.Printf("Was read %d bytes (total %d)\n", n, total)
			*/
			fw.Write(data[:n])
			chunks++
			fmt.Print(".")
		}
		fw.Close()
	}()

	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer f.Close()

	io_n, err := io.Copy(f, fr)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	fmt.Printf("\nread %d chunks in %d bytes\n", chunks, io_n)

}

func sendFile(yd IYaDisk, filename string) {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer f.Close()

	chunks := int64(0)
	pr, pw := io.Pipe()

	go func() {
		data := make([]byte, 1*1024*1024)
		for {
			n, err := f.Read(data)
			if err != nil {
				if err == io.EOF && n == 0 {
					break
				}
				log.Fatal(err)
			}
			pw.Write(data[:n])
			chunks++
			fmt.Print(".")
		}
		pw.Close()
	}()

	err = yd.PutData(filename, pr)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	fmt.Printf("\nread %d chunks\n", chunks)
}

func main() {

	if len(os.Args) < 2 {
		usage()
	}

	cmd := os.Args[1]

	yd := IYaDisk(yadisk.YaDisk(oAuthToken))

	switch cmd {
	default:
		usage()

	case "info":
		getInfo(yd)

	case "get":
		if len(os.Args) != 3 {
			usage()
		}

		filename := os.Args[2]

		getFile(yd, filename)

	case "curl":
		if len(os.Args) != 3 {
			usage()
		}

		filename := os.Args[2]

		getCurl(yd, filename)

	case "send":
		if len(os.Args) != 3 {
			usage()
		}

		filename := os.Args[2]

		sendFile(yd, filename)

	case "del":
		if len(os.Args) != 3 {
			usage()
		}

		path := os.Args[2]

		delRes(yd, path)

	}
}
