package templating

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

func mergeJson(configsnippets map[string]map[string]string) {
	results := map[string]string{}
	for nodename, nodedata := range configsnippets {
		results[nodename] = "{}"
		fmt.Printf("Node: %s\n", nodename)
		for snippetname, data := range nodedata {
			results[nodename] = runCommand(results[nodename], "{"+data+"}")
			fmt.Printf("%s %d\n", snippetname, len(results[nodename]))
		}
	}
	for x, y := range configsnippets["leaf1"] {
		f, _ := os.Create("/tmp/leaf1/" + x)
		f.WriteString("{" + y + "}")
		f.Close()
	}

	for x, y := range results {
		fmt.Printf("%s: %+v", x, string(y))
	}
	//fmt.Printf("%+v\n", results)
}

func runCommand(data1 string, data2 string) string {
	file1, err := ioutil.TempFile(os.TempDir(), "json-paco-")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	defer os.Remove(file1.Name())

	file2, err := ioutil.TempFile(os.TempDir(), "json-paco-")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	defer os.Remove(file2.Name())

	_, err = file1.WriteString(data1)
	if err != nil {
		log.Fatalf("%v", err)
	}

	_, err = file2.WriteString(data2)
	if err != nil {
		log.Fatalf("%v", err)
	}

	cmd := exec.Command("jq", "-s", ".[0] * .[1]", file1.Name(), file2.Name())
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		log.Fatalf("%v", err)
	}
	return out.String()
}

func initDoubleMap(data map[string]map[string]string, key1 string, key2 string) {
	if _, ok := data[key1]; !ok {
		data[key1] = map[string]string{}
	}
	if _, ok := data[key1][key2]; !ok {
		data[key1][key2] = ""
	}
}
