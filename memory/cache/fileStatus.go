package cache

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type PcStatus struct {
	Name      string    `json:"filename"`  // file cache as specified on command line
	Size      int64     `json:"size"`      // file size in bytes
	Timestamp time.Time `json:"timestamp"` // time right before calling mincore
	Mtime     time.Time `json:"mtime"`     // last modification time of the file
	Pages     int       `json:"pages"`     // total memory pages
	Cached    int       `json:"cached"`    // number of pages that are cached
	Uncached  int       `json:"uncached"`  // number of pages that are not cached
	Percent   float64   `json:"percent"`   // percentage of pages cached
	PPStat    []bool    `json:"status"`    // per-page status, true if cached, false otherwise
}

func GetPcStatus(fname string) (PcStatus, error) {
	pcs := PcStatus{Name: fname}
	f, err := os.Open(fname)
	if err != nil {
		return pcs, fmt.Errorf("could not open file for read: %v", err)
	}
	defer f.Close()

	// TEST TODO: verify behavior when the file size is changing quickly
	// while this function is running. I assume that the size parameter to
	// mincore will prevent overruns of the output vector, but it's not clear
	// what will be in there when the file is truncated between here and the
	// mincore() call.
	fi, err := f.Stat()

	if err != nil {
		return pcs, fmt.Errorf("could not stat file: %v", err)
	}
	if fi.IsDir() {
		return pcs, errors.New("file is a directory")
	}

	pcs.Size = fi.Size()
	pcs.Timestamp = time.Now()
	pcs.Mtime = fi.ModTime()
	pcs.PPStat, err = FileMincore(f, fi.Size())

	if err != nil {
		return pcs, err
	}

	// count the number of cached pages
	for _, b := range pcs.PPStat {
		if b {
			pcs.Cached++
		}
	}
	pcs.Pages = len(pcs.PPStat)
	pcs.Uncached = pcs.Pages - pcs.Cached

	// convert to float for the occasional sparsely-cached file
	// see the README.md for how to produce one
	pcs.Percent = (float64(pcs.Cached) / float64(pcs.Pages)) * 100.00
	return pcs, nil
}

func GetPidMaps(pid int) []string{
	fname := fmt.Sprintf("/proc/%d/maps",pid)
	fi,err:= os.Open(fname)
	if err != nil{
		log.Fatalf("could not open '%s' for read: %v", fname, err)
	}
	defer fi.Close()
	scanner := bufio.NewScanner(fi)
	var a  []string
	for scanner.Scan(){
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 6 && strings.HasPrefix(parts[5],"/") {
			a = append(a,parts[5])
		}
	}
	if err := scanner.Err(); err != nil{
		log.Fatalf("reading '%s' failed: %s", fname, err)
	}
	return a
}




