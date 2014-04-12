// vim: tabstop=4 softtabstop=4 shiftwidth=4 noexpandtab tw=72
//http://stackoverflow.com/questions/8757389/reading-file-line-by-line-in-go
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/atotto/clipboard"
	"os"
	"strconv"
	"strings"

	//	"io/ioutil"
)

type existsFunc func(string) bool

func osStatExists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

var ignoreList = [...]string{"/", ".", "./", "..", "../"}

//var rootListing, _ = ioutil.ReadDir("/")
//var pwdListing, _ = ioutil.ReadDir(".")

func ignored(file string) bool {
	for _, val := range ignoreList {
		if file == val {
			return true
		}
	}
	return false
}

func longestFileEndIndex(line []rune, exists existsFunc) int {
	// Possible optimisations:
	// 1. this should start at the end - the longest substring first
	// 2. it could be a good strategy to list files and try to
	//    find a common prefix - if not just stop right there and then
	//    from / if it starts with / and if not from `pwd`
	//    do file listing from / and `pwd` only once
	//    need to consider relative dirs though which is annoying

	maxIndex := 0
	for i, _ := range line {
		slice := line[0 : i+1]
		file := string(slice)
		if !ignored(file) {
			if exists(file) {
				// TODO if this is not a dir, stop here
				maxIndex = i
			}
		}
	}
	return maxIndex
}

func longestFileInLine(line string, exists existsFunc) (firstCharIndex int, lastCharIndex int) {
	for searchStartIndex, _ := range line {
		searchSpace := []rune(line[searchStartIndex:len(line)])
		lastCharIndexInSlice := longestFileEndIndex(searchSpace, exists)
		lastCharIndexInLine := lastCharIndexInSlice + searchStartIndex
		if lastCharIndexInSlice > 0 && lastCharIndexInLine > lastCharIndex {
			lastCharIndex = lastCharIndexInLine
			firstCharIndex = searchStartIndex
		}
	}

	return firstCharIndex, lastCharIndex
}

func askUser() (requestedNumbers []string, err error) {
	fmt.Println()
	fmt.Print("to clipboard: ")
	ttyFile, err := os.Open("/dev/tty")
	if err != nil {
		return nil, err
	}
	defer ttyFile.Close()
	ttyReader := bufio.NewReader(ttyFile)
	s, err := ttyReader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	return strings.Fields(s), nil
}

type processFunc func(string, int, int)

func printProcessor(fileCount *int) processFunc {
	return func(line string, firstCharIndex int, lastCharIndex int) {
		//file := line[firstCharIndex : lastCharIndex+1]
		fmt.Println(strconv.Itoa(*fileCount), line[:len(line)-1])
	}
}

func clipboardProcessor(clip *bytes.Buffer, fileCount *int) processFunc {
	argsWithoutProg := os.Args[1:]
	return func(line string, firstCharIndex int, lastCharIndex int) {
		file := line[firstCharIndex : lastCharIndex+1]

		// collect any file position arguments to copy to the
		// clipboard later
		for _, v := range argsWithoutProg {
			n, _ := strconv.Atoi(v)
			if n == *fileCount {
				clip.WriteString(file)
				clip.WriteString(" ")
			}
		}
	}
}

func fileNameCollectingProcessor(files *[]string) processFunc {
	return func(line string, firstCharIndex int, lastCharIndex int) {
		file := line[firstCharIndex : lastCharIndex+1]
		*files = append(*files, file)
	}
}

func main() {
	fileCount := 0
	var clip bytes.Buffer
	var files []string

	processors := []processFunc{
		printProcessor(&fileCount),
		clipboardProcessor(&clip, &fileCount),
		fileNameCollectingProcessor(&files),
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		firstCharIndex, lastCharIndex := longestFileInLine(line, osStatExists)

		if lastCharIndex > 0 {
			fileCount++
			for _, p := range processors {
				p(line, firstCharIndex, lastCharIndex)
			}
		} else {
			fmt.Print(line)
		}
	}

	requestedNumbers, err := askUser()
	if err != nil {
		fmt.Printf("failed to read input: %s\n", err)
		return
	}

	for _, n := range requestedNumbers {
		i, _ := strconv.Atoi(n)
		clip.WriteString(files[i-1])
		clip.WriteString(" ")
	}

	clipboardOutput := clip.String()
	if clipboardOutput != "" {
		clipboard.WriteAll(clipboardOutput)
	}
}
