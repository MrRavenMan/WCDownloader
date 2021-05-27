package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	InfoColor    = "\033[1;34m%s\033[0m"
	NoticeColor  = "\033[1;36m%s\033[0m"
	WarningColor = "\033[1;33m%s\033[0m"
	ErrorColor   = "\033[1;31m%s\033[0m"
	DebugColor   = "\033[0;36m%s\033[0m"
)

type Update struct {
	date   string
	path   string
	delete string
}

type Item struct {
	Name         string `json:"name"`
	Size         int    `json:"size"`
	Path         string `json:"path"`
	Download_url string `json:"download_url"`
	Type         string `json:"type"`
}

const download_url string = "https://raw.githubusercontent.com/drumbart/VFA-27_Ready_Room/master/"
const total_download_size int = 4000

var updateChart []Update

var bytes_downloaded int64 = 0
var total_size int = 0
var files_downloaded int = 0

func main() {
	// Before run info
	fmt.Println("Welcome to the Wildcats File Downloader!")
	fmt.Println("\nMake sure the script is in your .../Saved Games/DCS folder and your DCS is CLOSED")
	fmt.Println("NOTE: This might take 5-10 minutes the first time downloading all the files")
	fmt.Println("Make sure your DCS is CLOSED!")
	countdown(2)

	fmt.Println("Starting download of Wildcats files")

	updateChart = getUpdateChart()
	get_skins("https://api.github.com/repos/drumbart/VFA-27_Ready_Room/contents/Liveries")

	// Download complete info
	fmt.Println("\n ----------------------------------------")
	fmt.Printf(NoticeColor, "Download completed\n")
	fmt.Printf("%d files downloaded \n", files_downloaded)
	fmt.Printf("%d Mb downloaded \n", (bytes_downloaded / 1048576))
	fmt.Printf(WarningColor, "\n Press the Enter Key to stop anytime")
	fmt.Scanln()

}

func get_skins(url string) {

	var items []Item
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	json.Unmarshal(body, &items)

	for i := 0; i < len(items); i++ {
		// fmt.Println(items[i].Path)
		// fmt.Println(items[i].Type)
		fmt.Println()

		// Check file type of linked file - Exclude some file types from download
		switch {
		case items[i].Name == ".gitattributes":
			continue
		case items[i].Name == "wcmap.txt":
			continue
		case items[i].Name == "wcmapper.ps1":
			continue
		default:
			if items[i].Type == "dir" {
				// If directory does not exist already, make it
				if _, err := os.Stat(items[i].Path); os.IsNotExist(err) {
					os.Mkdir(items[i].Path, 0755)
				}
				if err != nil {
					log.Fatal(err)
				}
				// Query a list of all items inside the dir and start the process within the dir
				get_skins(fmt.Sprintf("https://api.github.com/repos/drumbart/VFA-27_Ready_Room/contents/%s", items[i].Path))

			} else { // If file type is not directory, check if file exists already in folder, otherwise download
				checkFile(items[i])
			}
			// Display status of script
			total_size += items[i].Size
			// fmt.Printf("Completed %d", ((total_size / 1048576) / total_download_size * 100))
		}
	}
}

func getUpdateChart() []Update {
	chart := make([]Update, 0, 4)

	var fileName string = "update_charts.txt"

	file, e := os.Open(fileName)
	if e != nil {
		fmt.Println("Error is = ", e)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s := strings.Split(scanner.Text(), " ")
		var update Update
		update.date, update.path, update.delete = s[0], s[1], s[2]
		chart = append(chart, update)
	}
	file.Close()

	return chart
}

func checkFile(item Item) {
	if _, err := os.Stat(item.Path); err == nil { // First check if file is already downloaded

		fileIn, chartIdx := fileInUpdateChart(item.Path)
		// Check if file is in update chart
		if fileIn == true { // File already exists and needs to be checked in update chart

			if updateChart[chartIdx].delete == "true" { // Check if file is set to be removed
				err := os.Remove(item.Path)
				if err != nil {
					fmt.Println(err)
				}
				fmt.Printf("Removing %s \n", item.Name)

			} else { // If file is not set to be removed, check if up to date

				// get last modified time from file on pc
				file, err := os.Stat(item.Path)
				if err != nil {
					fmt.Println(err)
				}
				modifiedtime := file.ModTime()

				updateTime, err := strconv.ParseInt(updateChart[chartIdx].date, 10, 64) // Convert date from update chart to time obj
				if err != nil {
					panic(err)
				}
				updatededTime := time.Unix(updateTime, 0)

				if updatededTime.Before(modifiedtime) { // Check if file is up to date
					err := os.Remove(item.Path) // If file is not up to date, remove and download new one
					if err != nil {
						fmt.Println(err)
					}
					fmt.Printf("File %s is outdated, downloading newest version \n", item.Name)
					e := download_file(item.Path, item.Size)
					if e != nil {
						panic(e)
					}
					fmt.Printf("Downloaded: %s - %d", item.Name, (item.Size / 1048576))
				} else {
					fmt.Printf("File %s already exists and is up to date! \n", item.Name)
				}
			}
		}
	} else { // File does not already exist and needs to be downloaded
		fmt.Printf("Downloading %s at %s \n", item.Name, item.Path)
		e := download_file(item.Path, item.Size)
		if e != nil {
			panic(e)
		}
		fmt.Printf("Downloaded: %s - %d", item.Name, (item.Size / 1048576))
	}
}

func fileInUpdateChart(path string) (bool, int) {
	for i, v := range updateChart {
		if v.path == path {
			return true, i
		}
	}
	var i int = 0
	return false, i
}

// Downlooad file from url and put  innto path
func download_file(path string, size int) error {

	// Get the data
	resp, err := http.Get((download_url + path))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)

	bytes_downloaded += int64(size)
	files_downloaded += 1
	return err
}

func countdown(count int) {
	fmt.Print("Starting download in ")
	for count > 0 {
		if rand.Intn(100) == 1 {
			break
		}
		fmt.Printf("%v...\n", count)
		time.Sleep(time.Second)
		count--
	}
}
