package cloudapihandler

import (
	"binehub/filesys"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type Type int

const (
	Downloaded Type = iota
	Deleted
	Provisioned
	FilterUpdated
)

type ApiData struct {
	Id            string
	ApiType       Type
	RetryCount    int
	OperationTime time.Time
	Properties    []Property
}

type Property struct {
	Key   string
	Value string
}

type ContentData struct {
	ContentId     string    `json:"contentId"`
	OperationTime time.Time `json:"operationTime"`
}

type UpdateRequest struct {
	DeviceId string        `json:"deviceId"`
	Contents []ContentData `json:"contents"`
}

type ApiDatas []ApiData

var fs *filesys.FileSystem
var baseUrl string
var deviceId string

func InitAPIHandler(filesystem filesys.FileSystem, burl string, devId string) {
	fs = &filesystem
	baseUrl = burl
	deviceId = devId
}

func HandleApiRequests(interval int) {
	var apiData []ApiData

	var result [][]byte

	for {
		fmt.Println("Processing pending api requests")
		log.Println("Processing pending api requests.")

		result = fs.GetPendingApiRequests()

		for _, data := range result {
			var apidata = new(ApiData)
			json.Unmarshal(data, apidata)

			apiData = append(apiData, *apidata)
		}

		var segragatedData = GetSegregatedItems(apiData)

		ProcessRequests(segragatedData)

		log.Printf("Done processing batch. Sleeping for %v minutes", interval)
		time.Sleep(time.Duration(interval) * time.Minute)
	}
}

func GetSegregatedItems(apiData []ApiData) map[Type][]ApiData {
	segragatedData := make(map[Type][]ApiData)

	var apiType Type

	for _, data := range apiData {
		apiType = data.ApiType
		var values = segragatedData[apiType]
		segragatedData[apiType] = append(values, data)
	}

	return segragatedData
}

func ProcessRequests(segragatedMap map[Type][]ApiData) {
	for key := range segragatedMap {
		switch key {
		case Downloaded:
			fmt.Println("Handle batch download request")
			handleBatchDownloadRequest(segragatedMap[key])
		case Deleted:
			fmt.Println("Handle batch delete request")
			handleBatchDeletedRequest(segragatedMap[key])
		case FilterUpdated:
			fmt.Println("Handle filter update request")
			handleFilterUpdatedRequest(segragatedMap[key])
		case Provisioned:
			fmt.Println("Handle provisioned request")
			handledProvisionedRequest(segragatedMap[key])
		}
	}
}

func handleBatchDownloadRequest(apidata []ApiData) {

	//base url
	//post req body with content ids
	//call api
	//proces response and add back failed ones in db

	url := baseUrl + "api/v1/DeviceContent/downloaded"

	var contentData []ContentData

	for _, data := range apidata {

		content := new(ContentData)
		content.ContentId = data.Id
		content.OperationTime = data.OperationTime

		contentData = append(contentData, *content)
	}

	updateRequest := new(UpdateRequest)
	updateRequest.DeviceId = deviceId
	updateRequest.Contents = contentData

	body, err := json.Marshal(updateRequest)

	if err != nil {
		log.Printf("[handleBatchDownloadRequest] Error in marshalling of request body %v", err)
		return
	}

	requestBody := bytes.NewBuffer(body)

	resp, err := http.Post(url, "application/json", requestBody)

	if err != nil {
		log.Printf("[handleBatchDownloadRequest] error in post request %v", err)
		return
	}

	defer resp.Body.Close()
	//Read the response body
	respbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}

	sb := string(respbody)
	log.Println(sb)

	err = fs.DeletePendingAPIRequestEntries(ApiDatas(apidata).GetIds())

	if err != nil {
		log.Printf("[handleBatchDownloadRequest] Error in deleting db entries %s", err)
	}

	//parse response
	// if there are failures, re add to bolt
}

func handleBatchDeletedRequest(apiData []ApiData) {

}

func handleFilterUpdatedRequest(apiData []ApiData) {

}

func handledProvisionedRequest(apiData []ApiData) {
	if len(apiData) > 1 {
		log.Printf("Found more than one entry for provision request %v", len(apiData))
	}

	url := baseUrl + "api/v1/Device/provision/"

	fmt.Println("Url " + url)

	for _, data := range apiData {
		devUrl := url + data.Id
		err := putRequest(devUrl, nil)

		if err != nil {
			log.Printf("Re queuing the request as it failed %v ", data.Id)
			byteData, err := json.Marshal(data)

			if err == nil {
				fs.AddProvisionedStatus(data.Id, byteData)
			} else {
				log.Printf("Failed to requeue provision request %v", err)
			}
		}
	}

	err := fs.DeletePendingAPIRequestEntries(ApiDatas(apiData).GetIds())

	if err != nil {
		log.Printf("[handledProvisionedRequest] Error in deleting db entries %s", err)
	}

}

func putRequest(url string, data io.Reader) error {
	fmt.Println("calling put request ")
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPut, url, data)
	var bearer = "Bearer " // + accesstoken todo: Get accesstoken from kaizala
	req.Header.Add("Authorization", bearer)

	if err != nil {
		log.Printf("Error in put request NewRequest %s", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error in put request Client.Do %s", err)
	}

	defer resp.Body.Close()
	respCode := resp.StatusCode

	if respCode != http.StatusNoContent {
		log.Printf("Put request did not succeed : %v ", respCode)
		return errors.New("failed_request")
	}

	return nil
}

func (apidatas ApiDatas) GetIds() []string {
	var ids []string

	for _, data := range apidatas {
		ids = append(ids, data.Id)
	}
	return ids
}
