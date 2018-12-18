package worker

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/graph-uk/combat-worker/models"
	//"github.com/mholt/archiver"
	"github.com/graph-uk/combat-worker/utils"
	resty "gopkg.in/resty.v1"
)

// CombatWorker ...
type CombatWorker struct {
	startPath string
	serverURL string
}

func (t *CombatWorker) getServerURLFromCLI() (string, error) {
	if len(os.Args) < 2 {
		return "", errors.New("Server URL is required")
	}
	return os.Args[1], nil
}

// NewCombatWorker ...
func NewCombatWorker() (*CombatWorker, error) {
	var result CombatWorker
	var err error

	result.startPath, err = os.Getwd()
	if err != nil {
		fmt.Println("Cannot get current filepath (pwd)")
		return &result, err
	}

	result.serverURL, err = result.getServerURLFromCLI()
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
	//archiver.Zip.Make(tmpFile.Name(), []string{"out"})
	utils.Zipit(`out`, tmpFile.Name())
	return tmpFile.Name()
}

func handleError(err error) (command models.Command, params string, caseID int) {
	fmt.Println()
	fmt.Printf("%s", err)
	return models.Idle, "", 0
}

func (t *CombatWorker) getJob(host string) (command models.Command, params string, caseID int) {

	url := fmt.Sprintf("%s/api/v1/jobs/acquire", host)

	resp, err := resty.R().
		Post(url)

	if err != nil {
		return handleError(err)
	}

	var model models.AcquireJobResponseModel

	if resp.StatusCode() == http.StatusNotFound {
		return models.Idle, "", 0
	}

	err = json.Unmarshal(resp.Body(), &model)

	if err != nil {
		return handleError(err)
	}

	caseContent, err := base64.StdEncoding.DecodeString(model.Content)

	if err != nil {
		return handleError(err)
	}

	ioutil.WriteFile("./job/archived.zip", caseContent, 0777)

	return models.RunCase, model.CommandLine, model.CaseID
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

func (t *CombatWorker) doRunCase(params string, caseID int) {
	fmt.Println("CaseRunning " + params)

	err := utils.Unzip(`./job/archived.zip`, `./job/unarch`)
	if err != nil {
		fmt.Println(err.Error())
	}
	os.Chdir("job/unarch/src/Tests")

	rootTestsPath, _ := os.Getwd()
	rootTestsPath += string(os.PathSeparator) + ".." + string(os.PathSeparator) + ".."

	command := []string{"run"}
	command = append(command, strings.Split(params, " ")...)
	os.Chdir(command[1])

	slash := string(os.PathSeparator)
	os.RemoveAll("out")
	err = os.Rename(".."+slash+".."+slash+".."+slash+".."+slash+".."+slash+"outBackUp", "out")
	if err != nil {
		fmt.Println(err.Error())

	}
	command[1] = "main.go"

	cmd := exec.Command("go", command...)
	cmd.Env = t.addToGOPath(rootTestsPath)
	var out bytes.Buffer
	var outErr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &outErr
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
	//return
}

func (t *CombatWorker) postCaseResult(caseID int, exitStatus, stdout string) error {
	var content string

	if _, err := os.Stat("out"); err != nil { // if we don't have "out" directory - create it
		os.MkdirAll("out", 0777)
	}

	//	files, err := ioutil.ReadDir(`out`)
	//	if err != nil {
	//		panic(err)
	//	}

	//	for _, f := range files {
	//		if !(strings.Contains(f.Name(), `.txt`) || strings.Contains(f.Name(), `.html`) || strings.Contains(f.Name(), `.png`)) {
	//			os.Remove(`out` + string(os.PathSeparator) + f.Name()) // hotfix for carousel
	//		}
	//	}
	//os.Remove(`out` + string(os.PathSeparator) + `SeleniumSessionID.txt`) // hotfix for carousel

	outFileName := t.packOutputToTemp()
	slash := string(os.PathSeparator)

	err := os.Rename("out", ".."+slash+".."+slash+".."+slash+".."+slash+".."+slash+"outBackUp")
	if err != nil {
		panic(err)
	}

	fileContent, err := ioutil.ReadFile(outFileName)

	if err != nil {
		return err
	}

	content = base64.StdEncoding.EncodeToString(fileContent)

	url := fmt.Sprintf("%s/api/v1/cases/%d/tries", t.serverURL, caseID)

	resp, err := resty.R().
		SetBody(&models.TryModel{
			Content:    content,
			ExitStatus: exitStatus,
			Output:     stdout}).
		Post(url)

	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("Incorrect request status: %d", resp.StatusCode())
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

// Process ...
func (t *CombatWorker) Process() {
	os.Chdir(t.startPath)
	cleanupJob()
	command, params, caseID := t.getJob(t.serverURL)
	if command == models.RunCase {
		t.doRunCase(params, caseID)
	}
	if command == models.Idle {
		time.Sleep(5 * time.Second)
	}
}
