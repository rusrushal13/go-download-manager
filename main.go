package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// Download struct for downloading the file
type Download struct {
	URL           string
	TargetPath    string
	TotalSections int
}

// Get new Request
func (d Download) getNewRequest(method string) (*http.Request, error) {
	r, err := http.NewRequest(
		method,
		d.URL,
		nil,
	)

	if err != nil {
		return nil, err
	}

	r.Header.Set("User-Agent", "Silly Download Manager v001")
	return r, nil
}

func (d Download) mergeFiles(sections [][2]int) error {
	targetFile, err := os.OpenFile(d.TargetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	for i := range sections {
		tmpFileName := fmt.Sprintf("section-%v.tmp", i)
		b, err := ioutil.ReadFile(tmpFileName)
		if err != nil {
			return err
		}
		bytesMerged, err := targetFile.Write(b)
		if err != nil {
			return err
		}
		err = os.Remove(tmpFileName)
		if err != nil {
			return err
		}
		fmt.Printf("%v bytes merged\n", bytesMerged)
	}
	return nil
}

func (d Download) downloadSection(tempFileIndex int, content [2]int) error {
	newGetRequest, err := d.getNewRequest("GET")
	if err != nil {
		return err
	}
	newGetRequest.Header.Set(
		"Range",
		fmt.Sprintf("bytes=%v-%v", content[0], content[1]),
	)
	responseForRequest, err := http.DefaultClient.Do(newGetRequest)
	if err != nil {
		return err
	}
	fmt.Printf("Downloaded %v bytes for section %v: %v\n", responseForRequest.Header.Get("Content-length"), tempFileIndex, content)

	b, err := ioutil.ReadAll(responseForRequest.Body)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(
		fmt.Sprintf("section-%v.tmp", tempFileIndex),
		b,
		os.ModePerm,
	)
	if err != nil {
		return err
	}
	return nil
}

// Do function for download manager
func (d Download) Do() error {
	fmt.Println(`Making connections`)

	r, err := d.getNewRequest("HEAD")
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	fmt.Printf("Got %v\n", resp.StatusCode)
	if resp.StatusCode > 299 {
		return fmt.Errorf("Can't process, response is %v", resp.StatusCode)
	}

	size, err := strconv.Atoi(resp.Header.Get("Content-length"))
	if err != nil {
		return err
	}
	fmt.Printf("Size is %v bytes\n", size)

	var sections = make([][2]int, d.TotalSections)
	eachSize := size / d.TotalSections + 1
	fmt.Printf("Each size is %v bytes\n", eachSize)

	for i := range sections {
		if i == 0 {
			sections[i][0] = 0
		} else {
			sections[i][0] = sections[i-1][1] + 1
		}

		if i < d.TotalSections-1 {
			sections[i][1] = sections[i][0] + eachSize
		} else {
			sections[i][1] = size - 1
		}
	}

	log.Println(sections)

	var wg sync.WaitGroup
	for i, section := range sections {
		wg.Add(1)
		i := i
		section := section
		go func() {
			defer wg.Done()
			err := d.downloadSection(i, section)
			if err != nil {
				panic(err)
			}
		}()
	}
	wg.Wait()

	return d.mergeFiles(sections)
}

func main() {
	startTime := time.Now()

	d := Download{
		URL:           ``,
		TargetPath:    "/tmp/targetFile",
		TotalSections: 10,
	}

	err := d.Do()
	if err != nil {
		log.Fatalf("An error occured while downloading the file: %s\n", err)
	}

	fmt.Printf("Download completed in %v seconds\n", time.Now().Sub(startTime).Seconds())
}
