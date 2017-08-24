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

// Flags struct contains the command line argument data
type Flags struct {
	FilePath  string
	File      *os.File
	FieldName string
	FieldID   string
}

// FieldData structs contain data on an individual field
type FieldData struct {
	Name   string
	ID     string
	Values []FieldValue
}

// FieldValue structs contain data on a value for a particular field
type FieldValue struct {
	Value string
	ID    string
}

func main() {
	// parse output file path
	f := &Flags{}
	flag.StringVar(&f.FilePath, "f", "", "Path to output file")
	flag.StringVar(&f.FieldName, "n", "", "A field to search for")
	flag.StringVar(&f.FieldID, "i", "", "A customfield ID to search for")
	flag.Parse()

	f.Validate()
	defer f.File.Close()

	// read creds in from command line
	r := bufio.NewReader(os.Stdin)
	blue := color.New(color.FgBlue)
	blue.Print("Enter JIRA username: ")
	un, _ := r.ReadString('\n')
	un = strings.TrimSuffix(un, "\n")

	blue.Print("Enter JIRA password: ")
	bytePW, _ := terminal.ReadPassword(int(syscall.Stdin))
	pw := string(bytePW)
	pw = strings.TrimSuffix(pw, "\n")

	blue.Print("\nEnter JIRA base URL: ")
	url, _ := r.ReadString('\n')
	url = strings.TrimSuffix(url, "\n")
	url = strings.TrimSuffix(url, "/") + RestEndpoint

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
			fd.Write(f.File)
			return
		}
		color.Red("Specified field '%s' not found", f.FieldName)
		os.Exit(1)
	}

	// provide output for a given ID, if specified
	if f.FieldID != "" {
		for _, v := range vals {
			if v.ID == f.FieldID {
				v.Write(f.File)
				return
			}
		}
		color.Red("Specified ID value '%s' not found", f.FieldID)
		os.Exit(1)
	}

	// print values from the map
	count := 0
	for _, fd := range vals {
		fd.Write(f.File)
		count++
		if count < len(vals) {
			WriteSeparator("===================", f.File)
		}
	}

	return
}

// Validate checks command line inputs and sets values needed to run
func (f *Flags) Validate() {
	// set file for writing
	if f.FilePath == "" {
		f.File = nil
	} else {
		if _, err := os.Stat(f.FilePath); os.IsNotExist(err) {
			f.File, _ = os.Create(f.FilePath)
		} else {
			f.File, _ = os.OpenFile(f.FilePath, os.O_WRONLY, 0644)
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

// FieldData.print prints out the content of a FieldData object in an
// easily readable format
func (fd FieldData) Write(f *os.File) {
	// output to stdout
	if f == nil {
		field := color.New(color.BgGreen)
		field.Printf("Field Name: %s (ID: %s)", fd.Name, fd.ID)
		color.Unset()
		fmt.Println()
		for j := range fd.Values {
			color.Green("\tValue Name: %s (ID: %s)\n", fd.Values[j].Value, fd.Values[j].ID)
		}
	} else {
		w := bufio.NewWriter(f)
		defer w.Flush()
		w.WriteString(fmt.Sprintf("Field Name: %s (ID: %s)\n", fd.Name, fd.ID))
		for j := range fd.Values {
			w.WriteString(fmt.Sprintf("\tValue Name: %s (ID: %s)\n", fd.Values[j].Value, fd.Values[j].ID))
		}
	}
}

// WriteSeparator writes a specified separator in between FieldData values
func WriteSeparator(sep string, f *os.File) {
	if f == nil {
		fmt.Println(sep)
	} else {
		w := bufio.NewWriter(f)
		defer w.Flush()
		w.WriteString(fmt.Sprintln(sep))
	}
}

// get the issuetypes within a particular project
func getTypes(p *sj.Json, vals map[string]FieldData) {
	for i := range p.Get("issuetypes").MustArray() {
		it := p.Get("issuetypes").GetIndex(i)
		getFields(it, vals)
	}
}

// get the fields within a particular issuetype
func getFields(it *sj.Json, vals map[string]FieldData) {
	for k := range it.Get("fields").MustMap() {
		if strings.HasPrefix(k, "customfield_") {
			field := it.Get("fields").Get(k)
			// get the allowed values for this field
			name := field.Get("name").MustString()
			if _, ok := vals[name]; !ok {
				values := getAllowedValues(field)
				fd := FieldData{Name: name, ID: k, Values: values}
				vals[name] = fd
			}
		}
	}
}

// get the allowed values for each field
func getAllowedValues(field *sj.Json) []FieldValue {
	fv := make([]FieldValue, 0)
	for i := range field.Get("allowedValues").MustArray() {
		value := field.Get("allowedValues").GetIndex(i)
		fv = append(fv, FieldValue{Value: value.Get("value").MustString(), ID: value.Get("id").MustString()})
	}
	return fv
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
