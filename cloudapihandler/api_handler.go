package cloudapihandler

import (
	"binehub/filesys"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	l "binehub/logger"
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
type ContentInfo struct {
	DownloadTime time.Time
	FolderPath   string
}
type UpdateRequest struct {
	DeviceId string        `json:"deviceId"`
	Contents []ContentData `json:"contents"`
}

type ProvisionDeviceRequest struct {
	DeviceId string `json:"deviceId"`
}
type CommandStatusRequest struct {
	DeviceId      string `json:"deviceId"`
	CommandId     string `json:"commandId"`
	IsFailed      bool   `json:"isFailed"`
	FailureReason string `json:"failureReason"`
}
type ApiDatas []ApiData

var fs *filesys.FileSystem
var deviceId string
var logger *l.Logger

func InitAPIHandler(filesystem filesys.FileSystem, devId string, log *l.Logger) {
	fs = &filesystem
	deviceId = devId
	logger = log
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

		var segregatedData = GetSegregatedItems(apiData)

		ProcessRequests(segregatedData)

		apiData = nil // making it nil so that in next batch old entries do not exist

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
			log.Println("Handle batch download request")
			handleBatchDownloadRequest(segragatedMap[key])
		case Deleted:
			log.Println("Handle batch delete request")
			handleBatchDeletedRequest(segragatedMap[key])
		case FilterUpdated:
			log.Println("Handle filter update request")
			handleFilterUpdatedRequest(segragatedMap[key])
		case Provisioned:
			log.Println("Handle provisioned request")
			handledProvisionedRequest(segragatedMap[key])
		}
	}
}

func handleBatchDownloadRequest(apidata []ApiData) {

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
		log.Printf("[handleBatchDownloadRequest] Error in marshalling request. Failed to send grpc  %s ", err)
		return
	}

	sm := new(l.MessageSubType)
	telemetryCommand := new(l.TelemetryCommandData)
	telemetryCommand.CommandName = l.ContentDownloaded
	telemetryCommand.CommandData = string(body)
	sm.TelemetryCommandData = *telemetryCommand

	err = logger.Log(l.TelemetryCommandMessage, sm)

	if err != nil {
		// re add entire batch with incremented retry count
		ApiDatas(apidata).IncrementRetryCount()

		err = fs.AddContents(ApiDatas(apidata).GetIds(), ApiDatas(apidata).GetDataArray())

		if err != nil {
			log.Printf("[handleBatchDownloadRequest] Error in re-adding batch to db %s", err)
		}

	} else {
		err = fs.DeletePendingAPIRequestEntries(ApiDatas(apidata).GetIds())

		if err != nil {
			log.Printf("[handleBatchDownloadRequest] Error in deleting db entries %s", err)
		}
	}
}

func handleBatchDeletedRequest(apidata []ApiData) {
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
		log.Printf("[handleBatchDeletedRequest] Error in marshalling request. Failed to send grpc  %s ", err)
		return
	}

	sm := new(l.MessageSubType)
	telemetryCommand := new(l.TelemetryCommandData)
	telemetryCommand.CommandName = l.ContentDeleted
	telemetryCommand.CommandData = string(body)
	sm.TelemetryCommandData = *telemetryCommand

	err = logger.Log(l.TelemetryCommandMessage, sm)

	if err != nil {
		// re add entire batch with incremented retry count
		ApiDatas(apidata).IncrementRetryCount()

		err = fs.AddContents(ApiDatas(apidata).GetIds(), ApiDatas(apidata).GetDataArray())

		if err != nil {
			log.Printf("[handleBatchDeletedRequest] Error in re-adding batch to db %s", err)
		}

	} else {
		err = fs.DeletePendingAPIRequestEntries(ApiDatas(apidata).GetIds())

		if err != nil {
			log.Printf("[handleBatchDeletedRequest] Error in deleting db entries %s", err)
		}
	}
}

func handleFilterUpdatedRequest(apidata []ApiData) {

	var deleteIds []string

	for _, data := range apidata {

		sm := new(l.MessageSubType)
		telemetryCommand := new(l.TelemetryCommandData)
		telemetryCommand.CommandName = l.CompleteCommand

		commandStatusReq := new(CommandStatusRequest)
		commandStatusReq.DeviceId = data.Id
		responseMap := make(map[string]string, 3)
		for _, property := range data.Properties {
			responseMap[property.Key] = property.Value
		}
		commandStatusReq.CommandId = responseMap["commandId"]
		isFailed, err := strconv.ParseBool(responseMap["isFailed"])
		if err != nil {
			log.Printf("Error in parsing bool value in complete command request %s", err)
			fmt.Printf("Error in parsing bool value in complete command request%s", err)
		}
		commandStatusReq.IsFailed = isFailed
		commandStatusReq.FailureReason = responseMap["failureReason"]
		fmt.Printf("command response body: ", commandStatusReq)
		commandStatusBytes, err := json.Marshal(commandStatusReq)
		if err != nil {
			log.Printf("Error in serializing command complete request %s", err)
			continue
		}
		telemetryCommand.CommandData = string(commandStatusBytes)
		fmt.Printf("string after serializing:: ", telemetryCommand.CommandData)
		sm.TelemetryCommandData = *telemetryCommand

		err = logger.Log(l.TelemetryCommandMessage, sm)
		if err != nil {
			log.Printf("Re queuing the request as it failed %v ", data.Id)
			data.RetryCount++ //increment retry count before re adding
			byteData, err := json.Marshal(data)

			if err == nil {
				log.Printf("Adding completecommand request to db key- %v and value- %v", data.Id, data)
				fs.AddCommandStatus(commandStatusReq.CommandId, byteData)
			} else {
				log.Printf("Failed to requeue completecommand request %v", err)
			}
		} else {
			deleteIds = append(deleteIds, commandStatusReq.CommandId) //delete entries from db for success ones
		}
	}
	log.Printf("[handleCompleteCommandRequest] Deleting successful api calls from db %v", deleteIds)
	// ensure to delete only completed items
	err := fs.DeletePendingAPIRequestEntries(deleteIds)

	if err != nil {
		log.Printf("[handleCompleteCommandRequest] Error in deleting db entries %s", err)
	}

}

func handledProvisionedRequest(apiData []ApiData) {
	if len(apiData) > 1 {
		log.Printf("Found more than one entry for provision request %v", len(apiData))
	}

	var deleteIds []string

	for _, data := range apiData {

		sm := new(l.MessageSubType)
		telemetryCommand := new(l.TelemetryCommandData)
		telemetryCommand.CommandName = l.ProvisionDevice

		provDevReq := new(ProvisionDeviceRequest)
		provDevReq.DeviceId = data.Id
		provReqBytes, err := json.Marshal(provDevReq)

		if err != nil {
			log.Printf("Error in serializing Provision device request %s", err)
			continue
		}

		telemetryCommand.CommandData = string(provReqBytes)
		sm.TelemetryCommandData = *telemetryCommand

		err = logger.Log(l.TelemetryCommandMessage, sm)

		if err != nil {
			log.Printf("Re queuing the request as it failed %v ", data.Id)
			data.RetryCount++ //increment retry count before re adding
			byteData, err := json.Marshal(data)

			if err == nil {
				log.Printf("Adding provisioned request to db key- %v and value- %v", data.Id, data)
				fs.AddProvisionedStatus(data.Id, byteData)
			} else {
				log.Printf("Failed to requeue provision request %v", err)
			}
		} else {
			deleteIds = append(deleteIds, data.Id) //delete entries from db for success ones
		}
	}

	log.Printf("[handledProvisionedRequest] Deleting successful api calls from db %v", deleteIds)
	// ensure to delete only completed items
	err := fs.DeletePendingAPIRequestEntries(deleteIds)

	if err != nil {
		log.Printf("[handledProvisionedRequest] Error in deleting db entries %s", err)
	}

}

func (apidatas ApiDatas) GetIds() []string {
	var ids []string

	for _, data := range apidatas {
		ids = append(ids, data.Id)
	}
	return ids
}

func (apidatas ApiDatas) GetDataArray() [][]byte {
	var dataArr [][]byte

	for _, data := range apidatas {
		bytedata, err := json.Marshal(data)

		if err != nil {
			log.Printf("Error in marshalling of apidata %s", err)
			continue
		}

		dataArr = append(dataArr, bytedata)
	}

	return dataArr
}

func (apidatas ApiDatas) IncrementRetryCount() {

	for _, data := range apidatas {
		data.RetryCount++
	}
}

func HandleAssetMapRequest(messageSize int) error {
	assetMap, err := fs.GetAssetInfoMapItems()
	if err != nil {
		log.Printf("Error in getting assetInfo map %s", err)
		fmt.Printf("Error in getting assetInfo map %s", err)
		return err
	}
	fmt.Println("AssetMap size::", len(assetMap))
	log.Println("AssetMap size::", len(assetMap))
	contentBatch := make([]ContentData, 0, messageSize)
	for id, v := range assetMap {
		contentInfo := ContentInfo{}
		err = json.Unmarshal(v, &contentInfo)
		if err != nil {
			log.Printf("Error in marshalling the contentinfo while sending the assetmap %s", err)
			return err
		}
		content := new(ContentData)
		content.ContentId = id
		content.OperationTime = contentInfo.DownloadTime

		contentBatch = append(contentBatch, *content)
		if len(contentBatch) == messageSize {
			//send the map
			// empty the contentbatch
			updateRequest := new(UpdateRequest)
			updateRequest.DeviceId = deviceId
			updateRequest.Contents = contentBatch
			err = sendDownloadedTelemetryData(*updateRequest)
			if err != nil {
				log.Printf("[handleAssetMapRequest] Error in marshalling request. Failed to send grpc  %s ", err)
				return err
			}
			contentBatch = contentBatch[:0]
		}
	}
	if len(contentBatch) > 0 {
		//send the remaining assets
		updateRequest := new(UpdateRequest)
		updateRequest.DeviceId = deviceId
		updateRequest.Contents = contentBatch
		err = sendDownloadedTelemetryData(*updateRequest)
		if err != nil {
			log.Printf("[handleAssetMapRequest] Error in marshalling request. Failed to send grpc  %s ", err)
			return err
		}
	}
	return nil
}

func sendDownloadedTelemetryData(updateRequest UpdateRequest) error {
	body, err := json.Marshal(updateRequest)

	if err != nil {
		return err
	}

	sm := new(l.MessageSubType)
	telemetryCommand := new(l.TelemetryCommandData)
	telemetryCommand.CommandName = l.ContentDownloaded
	telemetryCommand.CommandData = string(body)
	sm.TelemetryCommandData = *telemetryCommand
	err = logger.Log(l.TelemetryCommandMessage, sm)
	if err != nil {
		return err
	}
	return nil
}
