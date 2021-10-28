package cloudapihandler

import (
	"binehub/filesys"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	l "binehub/logger"

	"gopkg.in/ini.v1"
)

type Type int

const (
	Downloaded Type = iota
	Deleted
	Provisioned
	FilterUpdated
)
const (
	SendCloudTelemetry      = "sendCloudTelemetry"
	SendIOTCentralTelemetry = "sendIotCentralTelemetry"
)

type ApiData struct {
	Id                      string
	ApiType                 Type
	RetryCount              int
	OperationTime           time.Time
	SendCloudTelemetry      bool
	SendIOTCentralTelemetry bool
	Properties              []Property
}

type Property struct {
	Key   string
	Value string
}

type ContentData struct {
	ContentId     string    `json:"contentId"`
	OperationTime time.Time `json:"operationTime"`
}
type ContentInfoOnDevice struct {
	DownloadTime time.Time
	FolderPath   string
	CommandId    string
}
type ContentProperties struct {
	Size               float64   `json:"size,omitempty"`
	ContentId          string    `json:"contentId,omitempty"`
	ContainerLocation  string    `json:"containerLocation,omitempty"`
	OperationTime      time.Time `json:"operationTime, omitempty"`
	SesCID             string    `json:"sesCID,omitempty"`
	BroadcastCommandId string    `json:"broadcastCommandId,omitempty"`
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
var deviceCfg *ini.File
var deviceIniFilename string

func InitAPIHandler(filesystem filesys.FileSystem, devId string, log *l.Logger) {
	fs = &filesystem
	deviceId = devId
	logger = log

}
func InitAssetMapCall(file string, deviceIni *ini.File) {
	deviceIniFilename = file
	deviceCfg = deviceIni
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
	var contentProperties []ContentProperties
	for _, data := range apidata {
		if data.SendCloudTelemetry {
			content := new(ContentData)
			content.ContentId = data.Id
			content.OperationTime = data.OperationTime

			contentData = append(contentData, *content)
		}
		if data.SendIOTCentralTelemetry {
			content := new(ContentProperties)
			responseMap := make(map[string]string)
			content.ContentId = data.Id
			content.OperationTime = data.OperationTime
			for _, property := range data.Properties {
				responseMap[property.Key] = property.Value
			}
			content.BroadcastCommandId = responseMap["commandId"]
			content.ContainerLocation = responseMap["containerLocation"]
			content.SesCID = responseMap["sesCid"]
			content.Size, _ = strconv.ParseFloat(responseMap["size"], 64)

			contentProperties = append(contentProperties, *content)
		}

	}
	var telemetryCommandErr, iotCentralTelemetryErr error
	if len(contentData) > 0 {
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

		telemetryCommandErr = logger.Log(l.TelemetryCommandMessage, sm)

	}
	if len(contentProperties) > 0 {
		body, err := json.Marshal(contentProperties)

		if err != nil {
			log.Printf("[handleBatchDownloadRequest] Error in marshalling request for contentProperties. Failed to send grpc  %s ", err)
			return
		}

		sm := new(l.MessageSubType)
		contentInfo := new(l.ContentsInfo)
		contentInfo.NumberOfContents = len(contentProperties)
		contentInfo.ContentProperties = string(body)
		sm.ContentsInfo = *contentInfo
		iotCentralTelemetryErr = logger.Log(l.AssetDownloadOnDeviceFromSES, sm)
	}

	if telemetryCommandErr != nil || iotCentralTelemetryErr != nil {
		if telemetryCommandErr == nil {
			//flag for cloudTelemetry is set to false. Sent successfully
			ApiDatas(apidata).setTelemetryFlag(SendCloudTelemetry, false)
		}
		if iotCentralTelemetryErr == nil {
			// iotcetntral telemtry set to false.
			ApiDatas(apidata).setTelemetryFlag(SendIOTCentralTelemetry, false)

		}
		// re add entire batch with incremented retry count
		ApiDatas(apidata).IncrementRetryCount()

		err := fs.AddContents(ApiDatas(apidata).GetIds(), ApiDatas(apidata).GetDataArray())

		if err != nil {
			log.Printf("[handleBatchDownloadRequest] Error in re-adding batch to db %s", err)
		}

	} else {
		err := fs.DeletePendingAPIRequestEntries(ApiDatas(apidata).GetIds())

		if err != nil {
			log.Printf("[handleBatchDownloadRequest] Error in deleting db entries %s", err)
		}
	}
}

func handleBatchDeletedRequest(apidata []ApiData) {
	var contentData []ContentData
	var contentProperties []ContentProperties
	for _, data := range apidata {
		contentId := strings.TrimPrefix(data.Id, "DL_")
		if data.SendCloudTelemetry {
			content := new(ContentData)
			content.ContentId = contentId
			content.OperationTime = data.OperationTime

			contentData = append(contentData, *content)
		}
		if data.SendIOTCentralTelemetry {
			content := new(ContentProperties)
			responseMap := make(map[string]string)
			content.ContentId = contentId
			content.OperationTime = data.OperationTime
			for _, property := range data.Properties {
				responseMap[property.Key] = property.Value
			}
			content.BroadcastCommandId = responseMap["commandId"]
			content.SesCID = responseMap["sesCid"]

			contentProperties = append(contentProperties, *content)
		}

	}
	var telemetryCommandErr, iotCentralTelemetryErr error
	if len(contentData) > 0 {
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

		telemetryCommandErr = logger.Log(l.TelemetryCommandMessage, sm)

	}
	if len(contentProperties) > 0 {
		body, err := json.Marshal(contentProperties)

		if err != nil {
			log.Printf("[handleBatchDeletedRequest] Error in marshalling request. Failed to send grpc  %s ", err)
			return
		}
		sm := new(l.MessageSubType)
		contentInfo := new(l.ContentsInfo)
		contentInfo.NumberOfContents = len(contentProperties)
		contentInfo.ContentProperties = string(body)
		sm.ContentsInfo = *contentInfo
		iotCentralTelemetryErr = logger.Log(l.AssetDeleteOnDeviceByScheduler, sm)
	}

	if telemetryCommandErr != nil || iotCentralTelemetryErr != nil {
		if telemetryCommandErr == nil {
			//flag for cloudTelemetry is set to false. Sent successfully
			ApiDatas(apidata).setTelemetryFlag(SendCloudTelemetry, false)
		}
		if iotCentralTelemetryErr == nil {
			// iotcetntral telemtry set to false.
			ApiDatas(apidata).setTelemetryFlag(SendIOTCentralTelemetry, false)

		}
		// re add entire batch with incremented retry count
		ApiDatas(apidata).IncrementRetryCount()

		err := fs.AddContents(ApiDatas(apidata).GetIds(), ApiDatas(apidata).GetDataArray())

		if err != nil {
			log.Printf("[handleBatchDeletedRequest] Error in re-adding batch to db %s", err)
		}

	} else {
		err := fs.DeletePendingAPIRequestEntries(ApiDatas(apidata).GetIds())

		if err != nil {
			log.Printf("[handleBatchDownloadRequest] Error in deleting db entries %s", err)
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

func (apidatas ApiDatas) setTelemetryFlag(flag string, val bool) {
	switch flag {
	case SendCloudTelemetry:
		for _, data := range apidatas {
			data.SendCloudTelemetry = val
		}
	case SendIOTCentralTelemetry:
		for _, data := range apidatas {
			data.SendIOTCentralTelemetry = val
		}
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

func HandleAssetMapRequest(messageSize, sleepInterval int) {
	for {
		currentTimestamp := time.Now().Unix()
		lastTimestamp, err := deviceCfg.Section("DEVICE_DETAIL").Key("ASSETMAP_TIMESTAMP").Int64()
		if err != nil {
			fmt.Println("unable to getlast timestamp of assetmap sent from ini file: ", err)
			log.Println(fmt.Sprintf("last timestamp of assetmap sent from ini file: %v", err))
			lastTimestamp = 0
		}
		timeElapsedInHr := (currentTimestamp - lastTimestamp) / 60 / 60
		if timeElapsedInHr >= 24 {
			err := sendAssetmapInBatches(messageSize)
			if err != nil {
				fmt.Println("Error in sending the AssetMap: ", err)
				log.Println(fmt.Sprintf("Error in sending the AssetMap %v", err))
			} else {
				// persist the lastTimestamp in the ini file
				deviceCfg.Section("DEVICE_DETAIL").Key("ASSETMAP_TIMESTAMP").SetValue(strconv.FormatInt(currentTimestamp, 10))
				if writeErr := deviceCfg.SaveTo(deviceIniFilename); writeErr != nil {
					fmt.Printf("unable to store last timestamp of assetmap sent in the ini file: %v", writeErr)
					log.Println(fmt.Sprintf("unable to store last timestamp of assetmap sent in the ini file: %v", writeErr))
				}
			}
		}
		log.Printf("Done sending the assetmap. Sleeping for %v minute", sleepInterval)
		time.Sleep(time.Duration(sleepInterval) * time.Minute)
	}
}
func sendAssetmapInBatches(messageSize int) error {
	assetMap, err := fs.GetAssetInfoMapItems()
	if err != nil {
		log.Printf("Error in getting assetInfo map %s", err)
		fmt.Printf("Error in getting assetInfo map %s", err)
		return err
	}
	fmt.Println("AssetMap size::", len(assetMap))
	log.Println("AssetMap size::", len(assetMap))
	isAssetMap := true
	contentBatch := make([]ContentData, 0, messageSize)
	for id, v := range assetMap {
		contentInfo := ContentInfoOnDevice{}
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
			err = sendDownloadedTelemetryData(*updateRequest, isAssetMap)
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
		err = sendDownloadedTelemetryData(*updateRequest, isAssetMap)
		if err != nil {
			log.Printf("[handleAssetMapRequest] Error in marshalling request. Failed to send grpc  %s ", err)
			return err
		}
	}
	return nil
}
func sendDownloadedTelemetryData(updateRequest UpdateRequest, isAssetMap bool) error {
	body, err := json.Marshal(updateRequest)

	if err != nil {
		return err
	}

	sm := new(l.MessageSubType)
	telemetryCommand := new(l.TelemetryCommandData)
	telemetryCommand.CommandName = l.ContentDownloaded
	telemetryCommand.CommandData = string(body)
	if isAssetMap {
		telemetryCommand.IsAssetMap = isAssetMap
	}
	sm.TelemetryCommandData = *telemetryCommand
	err = logger.Log(l.TelemetryCommandMessage, sm)
	if err != nil {
		return err
	}
	return nil
}
