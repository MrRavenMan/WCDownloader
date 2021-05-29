package main

import (
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
	Date   string `json:"date"`
	Path   string `json:"path"`
	Delete bool   `json:"delete"`
}

type Item struct {
	Name         string `json:"name"`
	Size         int    `json:"size"`
	Path         string `json:"path"`
	Download_url string `json:"download_url"`
	Type         string `json:"type"`
}

type KneeboardCat struct {
	Name    string          `json:"name"`
	Path    string          `json:"parent"`
	SubCats []SubCategories `json:"subcat"`
}

type SubCategories struct {
	Name        string   `json:"name"`
	Default     bool     `json:"default"`
	Description string   `json:"description"`
	Files       []string `json:"files"`
}

type ConfigSub struct {
	Class       string
	Name        string
	Description string
	Download    bool
}

const download_url string = "https://raw.githubusercontent.com/drumbart/VFA-27_Ready_Room/master/"
const total_download_size int = 4000

var updateChart []Update

var bytes_downloaded int64 = 0
var total_size int = 0
var files_downloaded int = 0

var firstTime string

func main() {
	// Before run info
	fmt.Println("Welcome to the Wildcats File Downloader!")
	fmt.Println("\nMake sure the script is in your .../Saved Games/DCS folder and your DCS is CLOSED")
	fmt.Println("NOTE: This might take 5-10 minutes the first time downloading all the files")
	fmt.Println("Make sure your DCS is CLOSED!")

	fmt.Println("Is this your first time setting up? [y/N] ")
	fmt.Scanln(&firstTime)

	basicCheck()
	getKneeboards()

	countdown(2)

	fmt.Println("Starting download of Wildcats liveries")

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
				_ = update_check(items[i])
				// If directory does not exist already, make it
				if _, err := os.Stat(items[i].Path); os.IsNotExist(err) {
					os.Mkdir(items[i].Path, 0755)
					fmt.Printf("Making directory %s \n", items[i].Path)
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
	// Download update chart from github
	resp, e := http.Get("https://raw.githubusercontent.com/MrRavenMan/WCDownloader/main/update_chart.json")
	if e != nil {
		fmt.Println(e)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var chart []Update

	json.Unmarshal(body, &chart)

	return chart
}

func checkFile(item Item) {
	if strings.Contains(firstTime, "y") || strings.Contains(firstTime, "Y") {
		fmt.Printf("Downloading %s at %s \n", item.Name, item.Path)
		e := download_file(item.Path, item.Size)
		if e != nil {
			panic(e)
		}
		fmt.Printf("Downloaded: %s - %d MB \n", item.Name, (item.Size / 1048576))
	} else if _, err := os.Stat(item.Path); err == nil { // First check if file is already downloaded
		download := update_check(item)
		if download == true {
			e := download_file(item.Path, item.Size)
			if e != nil {
				panic(e)
			}
			fmt.Printf("Downloaded: %s - %d MB \n", item.Name, (item.Size / 1048576))
		}

	} else { // File does not already exist and needs to be downloaded
		fmt.Printf("Downloading %s at %s \n", item.Name, item.Path)
		e := download_file(item.Path, item.Size)
		if e != nil {
			panic(e)
		}
		fmt.Printf("Downloaded: %s - %d MB \n", item.Name, (item.Size / 1048576))
	}
}

func fileInUpdateChart(path string) (bool, int) {
	for i, v := range updateChart {
		if v.Path == path {
			return true, i
		}
	}
	var i int = 0
	return false, i
}

func update_check(item Item) bool {
	fileIn, chartIdx := fileInUpdateChart(item.Path)
	// Check if file is in update chart
	if fileIn == true { // File already exists and needs to be checked in update chart

		if updateChart[chartIdx].Delete == true { // Check if file is set to be removed
			err := os.RemoveAll(item.Path)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("Removing %s \n", item.Name)
			return false

		} else { // If file is not set to be removed, check if up to date

			// get last modified time from file on pc
			file, err := os.Stat(item.Path)
			if err != nil {
				fmt.Println(err)
			}
			modifiedtime := file.ModTime()

			updateTime, err := strconv.ParseInt(updateChart[chartIdx].Date, 10, 64) // Convert date from update chart to time obj
			if err != nil {
				panic(err)
			}
			updatededTime := time.Unix(updateTime, 0)

			if updatededTime.Before(modifiedtime) { // Check if file is up to date
				err := os.RemoveAll(item.Path) // If file is not up to date, remove and download new one
				if err != nil {
					fmt.Println(err)
				}
				fmt.Printf("File %s is outdated, downloading newest version \n", item.Name)
				return true
			} else {
				fmt.Printf("File %s already exists and is up to date! \n", item.Name)
				return false
			}
		}
	} else { // File already exist but is not in update chart
		fmt.Printf("File %s already exists and is up to date! \n", item.Name)
		return false
	}
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

func getKneeboards() {
	var categories []KneeboardCat
	var configFields []ConfigSub
	resp, err := http.Get("https://raw.githubusercontent.com/MrRavenMan/WCDownloader/main/Kneeboards.json")
	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	json.Unmarshal(body, &categories)

	for i := 0; i < len(categories); i++ { // Loop through and add all fields to config array
		for x := 0; x < len(categories[i].SubCats); x++ {
			var configField ConfigSub = ConfigSub{
				Class:       categories[i].Name,
				Name:        categories[i].SubCats[x].Name,
				Description: categories[i].SubCats[x].Description,
				Download:    categories[i].SubCats[x].Default,
			}
			configFields = append(configFields, configField)
		}
	}

	var c1 int
	var all bool = false
	fmt.Println("Which kneeboards do you wish to download")
	println("Press 0 to use default download kneeboards")
	println("Press 1 to download all kneeboards")
	println("Press 2 to configure which kneeboards to download")

	fmt.Scanln(&c1)

	switch {
	case c1 == 0:
		fmt.Println("Using default settings")
	case c1 == 1:
		fmt.Println("Downloading everything")
		all = true
	case c1 == 2:
		// Create a new config file if no exists
		if _, err := os.Stat("KneeboardConfig.json"); err == nil {
			file, _ := ioutil.ReadFile("KneeboardConfig.json") // Config files already exists, update it but keep settings
			var oldConfigFields []ConfigSub
			json.Unmarshal(file, &oldConfigFields)

			for i := 0; i < len(configFields); i++ {
				for x := 0; x < len(oldConfigFields); x++ {
					if oldConfigFields[x].Class == configFields[i].Class && oldConfigFields[x].Name == configFields[i].Name {
						configFields[i].Download = oldConfigFields[x].Download
					}
				}
			}

		}
		file, _ := json.MarshalIndent(configFields, "", " ")

		_ = ioutil.WriteFile("KneeboardConfig.json", file, 0644)

		fmt.Println("You can configure which kneeboards you wish to download in the KneeboardConfig.json in your .../Saved Games/DCS folder")
		fmt.Println("Change the Download to either true or false (No spaces or capital letters!) - please do not change anything else!")
		fmt.Println("Press enter when you have saved and closed the config file")
		fmt.Scanln()

		fmt.Println("")

	}

	if _, err := os.Stat("Kneeboard/"); os.IsNotExist(err) { // Create Kneeboard dir if not already exist
		os.Mkdir("Kneeboard/", 0755)
	}
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < len(categories); i++ {
		// If directory does not exist already, make it
		if _, err := os.Stat(categories[i].Path); os.IsNotExist(err) {
			os.Mkdir(categories[i].Path, 0755)
		}
		if err != nil {
			log.Fatal(err)
		}
		for x := 0; x < len(categories[i].SubCats); x++ {
			fmt.Printf("Downloading kneeboards for %s - %s \n", categories[i].Name, categories[i].SubCats[x].Name)
			for y := 0; y < len(configFields); y++ {
				if categories[i].Name == configFields[y].Class && categories[i].SubCats[x].Name == configFields[y].Name {
					if configFields[y].Download == true || all == true {
						for f := 0; f < len(categories[i].SubCats[x].Files); f++ {
							download_file((categories[i].Path + categories[i].SubCats[x].Files[f]), 0)
						}
					} else {
						for f := 0; f < len(categories[i].SubCats[x].Files); f++ {
							err := os.Remove((categories[i].Path + categories[i].SubCats[x].Files[f]))
							if err != nil {
								fmt.Println(err)
							}
							fmt.Printf("Removing kneeboard %s ", categories[i].SubCats[x].Files[f])
						}
					}
				}
			}
			fmt.Printf("Completed downloading kneeboards for %s - %s \n", categories[i].Name, categories[i].SubCats[x].Name)
		}
	}
}

func basicCheck() {
	os.Mkdir("Kneeboard", 0755)
	os.Mkdir("Liveries", 0755)
}
