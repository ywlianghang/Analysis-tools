package cache

import (
	"Analysis-tool/memory/cache/utils"
	"flag"
	"log"
	"os"
	"path"
	"sort"
)

var (
	pidFlag,   topFlag                          int
	terseFlag, nohdrFlag, jsonFlag, unicodeFlag bool
	plainFlag, ppsFlag, histoFlag, bnameFlag    bool
	sortFlag bool
	flagMap map[string]bool
)
//type PcStatusList []cache.PcStatus
func init() {
	// TODO: error on useless/broken combinations
	flag.IntVar(&pidFlag, "pid", 0, "show all open maps for the given pid")
	flag.IntVar(&topFlag, "top", 0, "show top x cached files")
	flag.BoolVar(&terseFlag, "terse", false, "show terse output")
	flag.BoolVar(&nohdrFlag, "nohdr", false, "omit the header from terse & text output")
	flag.BoolVar(&jsonFlag, "json", false, "return data in JSON format")
	flag.BoolVar(&unicodeFlag, "unicode", false, "return data with unicode box characters")
	flag.BoolVar(&plainFlag, "plain", false, "return data with no box characters")
	flag.BoolVar(&ppsFlag, "pps", false, "include the per-page status in JSON output")
	flag.BoolVar(&histoFlag, "histo", false, "print a simple histogram instead of raw data")
	flag.BoolVar(&bnameFlag, "bname", false, "convert paths to basename to narrow the output")
	flag.BoolVar(&sortFlag, "sort", false, "sort output by cached pages desc")
	flagMap = make(map[string]bool)
	flagMap["terseFlag"] = terseFlag
	flagMap["nohdrFlag"] = nohdrFlag
	flagMap["jsonFlag"] = jsonFlag
	flagMap["unicodeFlag"] = unicodeFlag
	flagMap["plainFlag"] = plainFlag
	flagMap["ppsFlag"] = ppsFlag
	flagMap["histoFlag"] = histoFlag
	flagMap["bnameFlag"] = bnameFlag
	flagMap["sortFlag"] = sortFlag
}
func uniqueSlice(slice *[]string) {
	found := make(map[string]bool)
	total := 0
	for i, val := range *slice {
		if _, ok := found[val]; !ok {
			found[val] = true
			(*slice)[total] = (*slice)[i]
			total++
		}
	}

	*slice = (*slice)[:total]
}
func getStatsFromFiles(files []string) PcStatusList {
	stats := make(PcStatusList, 0, len(files))
	for _, fname := range files {
		status, err := GetPcStatus(fname)
		if err != nil {
			log.Printf("skipping %q: %v", fname, err)
			continue
		}
		// convert long paths to their basename with the -bname flag
		// this overwrites the original filename in pcs but it doesn't matter since
		// it's not used to access the file again -- and should not be!
		if bnameFlag {
			status.Name = path.Base(fname)
		}
		stats = append(stats, status)
	}
	return stats
}
func top(top int) {
	p, err := utils.Processes()

	if err != nil {
		log.Fatalf("err: %s", err)
	}
	if len(p) <= 0 {
		log.Fatal("Cannot find any process.")
	}
	results := make([]utils.Process, 0, 50)

	for _, p1 := range p {
		if p1.RSS() != 0 {
			results = append(results, p1)
		}
	}
	var files []string
	for _, process := range results {
		SwitchMountNs(process.Pid())
		maps := GetPidMaps(process.Pid())
		files = append(files, maps...)
	}
	uniqueSlice(&files)
	stats := getStatsFromFiles(files)
	sort.Sort(PcStatusList(stats))
	topStats := stats[:top]
	FormatStats(topStats,flagMap)
}
func Cache(){
	flag.Parse()
	if topFlag != 0 {
		top(topFlag)
		os.Exit(0)
	}
	files := flag.Args()
	if pidFlag != 0{
		SwitchMountNs(pidFlag)
		maps := GetPidMaps(pidFlag)
		files = append(files, maps...)
	}
	if len(files) == 0 {
		flag.Usage()
		os.Exit(1)
	}
	stats := getStatsFromFiles(files)
	FormatStats(stats,flagMap)
}