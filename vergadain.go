package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	sj "github.com/bitly/go-simplejson"
	"github.com/fatih/color"
)

// RestEndpoint should be appended to the base URL provided by
// the user in order to form the base API URL
const RestEndpoint = "/rest/api/2"

var (
	un, pw, url string
)

// Flags struct contains the command line argument data
type Flags struct {
	CredsPath string
	CredsFile *os.File
	DataPath  string
	DataFile  *os.File
	FieldName string
	FieldID   string
}

func main() {
	// parse command line options
	f := &Flags{}
	flag.StringVar(&f.CredsPath, "c", "", "Path to credentials .yaml file (optional)")
	flag.StringVar(&f.DataPath, "f", "", "Path to output file (optional)")
	flag.StringVar(&f.FieldName, "n", "", "A field to search for (optional)")
	flag.StringVar(&f.FieldID, "i", "", "A customfield ID to search for (optional)")
	flag.Parse()
	f.Validate()
	if f.CredsFile != nil {
		defer f.CredsFile.Close()
	}
	if f.DataFile != nil {
		defer f.DataFile.Close()
	}

	// read creds in from command line if file is not nil
	if f.CredsFile == nil {
		r := bufio.NewReader(os.Stdin)
		blue := color.New(color.FgBlue)
		blue.Print("Enter JIRA username: ")
		un, _ = r.ReadString('\n')
		un = strings.TrimSuffix(un, "\n")

		blue.Print("Enter JIRA password: ")
		bytePW, _ := terminal.ReadPassword(int(syscall.Stdin))
		pw = string(bytePW)
		pw = strings.TrimSuffix(pw, "\n")

		blue.Print("\nEnter JIRA base URL: ")
		url, _ = r.ReadString('\n')
		url = strings.TrimSuffix(url, "\n")
		url = strings.TrimSuffix(url, "/") + RestEndpoint
	}

	// get all fields, output all customfield options and get the values
	jc := &http.Client{}
	req, err := http.NewRequest("GET", url+"/issue/createmeta?expand=projects.issuetypes.fields", nil)
	checkErr(err)
	req.Header.Add("Content-type", "application/json")
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprint(un + ":" + pw)))
	req.Header.Add("Authorization", fmt.Sprint("Basic "+auth))
	res, err := jc.Do(req)
	checkResponse(res, err)
	defer res.Body.Close()

	// convert response body data to simpleJSON object
	data, err := ioutil.ReadAll(res.Body)
	checkErr(err)
	json, err := sj.NewJson(data)
	checkErr(err)

	// create map for storing values
	vals := make(map[string]FieldData, 1)

	// iterate over the JIRA response data
	for i := range json.Get("projects").MustArray() {
		p := json.Get("projects").GetIndex(i)
		getTypes(p, vals)
	}

	// provide the output for a given name, if specified
	if f.FieldName != "" {
		if fd, ok := vals[f.FieldName]; ok {
			fd.Write(f.DataFile)
			return
		}
		color.Red("Specified field '%s' not found", f.FieldName)
		os.Exit(1)
	}

	// provide output for a given ID, if specified
	if f.FieldID != "" {
		for _, v := range vals {
			if v.ID == f.FieldID {
				v.Write(f.DataFile)
				return
			}
		}
		color.Red("Specified ID value '%s' not found", f.FieldID)
		os.Exit(1)
	}

	// print values from the map
	count := 0
	for _, fd := range vals {
		fd.Write(f.DataFile)
		count++
		if count < len(vals) {
			WriteSeparator("===================", f.DataFile)
		}
	}

	return
}

// Validate checks command line inputs and sets values needed to run
func (f *Flags) Validate() {
	// read creds file
	if f.CredsPath != "" {
		if _, err := os.Stat(f.CredsPath); os.IsNotExist(err) {
			color.Red("error: supplied creds file does not exist")
			flag.Usage()
			os.Exit(1)
		}
		f.CredsFile, _ = os.OpenFile(f.CredsPath, os.O_RDONLY, 0644)
		credsBytes, err := ioutil.ReadAll(f.CredsFile)
		if err != nil {
			color.Red("error: couldn't read supplied creds file")
			os.Exit(1)
		}
		credsJSON, err := sj.NewJson(credsBytes)
		if err != nil {
			color.Red("error: couldn't parse creds JSON file")
		}
		un = credsJSON.Get("username").MustString()
		pw = credsJSON.Get("password").MustString()
		url = strings.TrimSuffix(credsJSON.Get("site_url").MustString(), "/") + RestEndpoint
		fmt.Println(url)
		if un == "" || pw == "" || url == "" {
			color.Red("error: creds file missing at least one of username, password, or site_url values")
			os.Exit(1)
		}
	}

	// set file for writing
	if f.DataPath != "" {
		if _, err := os.Stat(f.DataPath); os.IsNotExist(err) {
			f.DataFile, _ = os.Create(f.DataPath)
		} else {
			f.DataFile, _ = os.OpenFile(f.DataPath, os.O_WRONLY, 0644)
		}
	}

	// if name and id are specified, error
	if f.FieldName != "" && f.FieldID != "" {
		color.Red("Please specify either name or ID (not both)")
		os.Exit(1)
	}

	// provide the output for a given ID, if specified
	if f.FieldID != "" {
		if !strings.HasPrefix(f.FieldID, "customfield_") {
			_, err := strconv.Atoi(f.FieldID)
			if err != nil {
				color.Red("Specified ID value '%s' is invalid", f.FieldID)
				os.Exit(1)
			}
			f.FieldID = "customfield_" + f.FieldID
		}
	}
}

// checkErr is a basic error checking function, panicing on a
// non-nil error value
func checkErr(e error) {
	if e != nil {
		panic(e)
	}
}

// checkResponse checks an error, along with the associated
// HTTP response status code, and panics on any errors (Go or HTTP)
func checkResponse(res *http.Response, e error) {
	if e != nil || res == nil {
		panic(e)
	}
	if res.StatusCode > 299 {
		data, _ := ioutil.ReadAll(res.Body)
		panic(fmt.Errorf("ERROR: reponse status code %d - %s", res.StatusCode, string(data)))
	}
}
