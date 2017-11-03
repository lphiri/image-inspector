package insights

import (
	"fmt"
	"log"
	"time"
	"net/http"
        "io/ioutil"
	"bytes"
	"github.com/fsouza/go-dockerclient"

	"github.com/openshift/image-inspector/pkg/api"
)

const ScannerName = "redhatinsights"


type ScanRequest struct {
	ContentPath string
	ImageId     string
}

type InsightsScanner struct {
	ServerPort int
}

var _ api.Scanner = &InsightsScanner{}

func NewScanner(port int) (api.Scanner, error) {

	return &InsightsScanner{
		ServerPort: port,
	}, nil
}

// Scan will scan the image
func (s *InsightsScanner) Scan(path string, image *docker.Image) ([]api.Result, interface{}, error) {
	scanResults := []api.Result{}
	scanStarted := time.Now()
	defer func() {
		log.Printf("Insights scan took %ds (%d actions found)", int64(time.Since(scanStarted).Seconds()), len(scanResults))
	}()
	scanResults, err := s.ScanImage(path,image.ID)
	if err != nil {
		return nil, nil, err
	}

	return scanResults, nil, nil
}

func (s *InsightsScanner) Name() string {
	return ScannerName
}





func (s *InsightsScanner) ScanImage(path string, id string) ([]api.Result,error){

	var netClient = &http.Client{
		Timeout: time.Second * 30,
	}
	var jsonreq string = `{"ContentPath":"`+path+`","ImageId":"`+id+`"}`
	var jsonStr = []byte(jsonreq)
	req, err := http.NewRequest("POST", "http://localhost:" + fmt.Sprint(s.ServerPort)+"/inspect", bytes.NewBuffer(jsonStr))
	if err != nil {
		fmt.Println("Unable to create request.")
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := netClient.Do(req)
	if err != nil {
		fmt.Println("Unable to reach the server.")
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if (resp.StatusCode != 200){
		fmt.Println("Error response from server " + string(body))
		return nil, err
	}

	//fmt.Println("body=", string(body))
	scanResults := []api.Result{}
	//for _, r := range clamResults.Files {
	//	r := api.Result{
	//		Name:           ScannerName,
	//		ScannerVersion: "3.0",
	//		Timestamp:      scanStarted,
	//		Reference:      fmt.Sprintf("file://%s", strings.TrimPrefix(r.Filename, path)),
	//		Description:    r.Result,
	//	}
	//	scanResults = append(scanResults, r)
	//}
	return scanResults,nil
}
