package combatWorker

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type CombatWorker struct {
	startPath string
	serverURL string
}

func (t *CombatWorker) getServerUrlFromCLI() (string, error) {
	if len(os.Args) < 2 {
		return "", errors.New("Server URL is required")
	}
	return os.Args[1], nil
}

func NewCombatWorker() (*CombatWorker, error) {
	var result CombatWorker
	var err error

	result.startPath, err = os.Getwd()
	if err != nil {
		fmt.Println("Cannot get current filepath (pwd)")
		return &result, err
	}

	result.serverURL, err = result.getServerUrlFromCLI()
	if err != nil {
		return &result, err
	}

	return &result, nil
}

func (t *CombatWorker) packOutputToTemp() string {
	fmt.Println("Packing output")
	tmpFile, err := ioutil.TempFile("", "combatOutput")
	if err != nil {
		panic(err)
	}
	tmpFile.Close()
	zipit("out", tmpFile.Name())
	return tmpFile.Name()
}

func (t *CombatWorker) getJob(host string) (command string, params string, sessionID string) {
	response, err := http.Post(host+"/api/v1/jobs/acquire", "application/json", bytes.NewBuffer([]byte{}))
	if err != nil {
		fmt.Println()
		fmt.Printf("%s", err)
		return "idle", "", ""
	} else {
		fmt.Println("getJob - " + response.Header.Get("command"))
		defer response.Body.Close()
		command = response.Header.Get("Command")
		if command == "idle" {
			return command, "", ""
		}
		params = response.Header.Get("Params")
		sessionID = response.Header.Get("SessionID")
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Println("Cannot read request body")
			fmt.Println(err.Error())
		}
		ioutil.WriteFile("./job/archived.zip", contents, 0777)
	}
	return command, params, sessionID
}

func (t *CombatWorker) addToGOPath(pathExtention string) []string {
	result := os.Environ()
	for curVarIndex, curVarValue := range result {
		if strings.HasPrefix(curVarValue, "GOPATH=") {
			result[curVarIndex] = result[curVarIndex] + string(os.PathListSeparator) + pathExtention
			return result
		}
	}
	return result
}

func (t *CombatWorker) doRunCase(params, caseID string) {
	fmt.Println("CaseRunning " + params)
	err := unzip("./job/archived.zip", "./job/unarch")
	if err != nil {
		fmt.Print(err.Error())
	}
	os.Chdir("job/unarch/src/Tests")
	//os.Exit(0)
	rootTestsPath, _ := os.Getwd()
	rootTestsPath += string(os.PathSeparator) + ".." + string(os.PathSeparator) + ".."
	//	fmt.Println(t.addToGOPath(rootTestsPath))
	//	os.Exit(0)

	command := []string{"run"}
	//fmt.Println(params)
	command = append(command, strings.Split(params, " ")...)
	os.Chdir(command[1])
	command[1] = "main.go"

	cmd := exec.Command("go", command...)
	cmd.Env = t.addToGOPath(rootTestsPath)
	var out bytes.Buffer
	var outErr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &outErr
	//	fmt.Println(command)
	fmt.Print("Run case... ")
	exitStatus := cmd.Run()

	exitStatusString := ""
	if exitStatus == nil {
		exitStatusString = "0"
		fmt.Println("Ok")
	} else {
		exitStatusString = exitStatus.Error()
		fmt.Println("Fail")
		fmt.Println(out.String())
		fmt.Println(outErr.String())
	}

	t.postCaseResult(caseID, exitStatusString, out.String()+outErr.String())
	return
}

func (t *CombatWorker) postCaseResult(caseID, exitStatus, stdout string) error {
	var content string

	if exitStatus != "0" { // send out only while try is failed
		if _, err := os.Stat("out"); err != nil { // if we have has not "out" directory - create it
			os.MkdirAll("out", 0777)
		}
		outFileName := t.packOutputToTemp()

		fileContent, err := ioutil.ReadFile(outFileName)

		if err != nil {
			return err
		}

		content = base64.StdEncoding.EncodeToString(fileContent)
	}

	json := fmt.Sprintf("{\"content\": \"%s\", \"exitStatus\":\"%s\", \"output\":\"%s\"}", content, exitStatus, stdout)
	body := bytes.NewBuffer([]byte(json))

	resp, err := http.Post(fmt.Sprintf("%s/api/v1/cases/%s/tries", t.serverURL, caseID), "application/json", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Println(resp.Status)
	if resp.StatusCode != 200 {
		return errors.New("Incorrect request status: " + strconv.Itoa(resp.StatusCode))
	}
	return nil
}

func cleanupJob() error {
	os.RemoveAll("job")

	time.Sleep(1 * time.Second)
	err := os.MkdirAll("job", 0777)
	if err != nil {
		fmt.Println("Cannot create job folder")
		return err
	}
	return nil
}

func (t *CombatWorker) Work() {
	os.Chdir(t.startPath)
	cleanupJob()
	command, params, sessionID := t.getJob(t.serverURL)
	if command == "RunCase" {
		t.doRunCase(params, sessionID)
	}
	if command == "idle" {
		time.Sleep(5 * time.Second)
	}
}
