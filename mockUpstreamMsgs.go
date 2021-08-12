package main

import (
	"math/rand"
	"time"

	l "binehub/logger"
)

// func mock_liveness(interval int) {
// 	for true {
// 		sm := new(l.SubValueType)
// 		sm.StringMessage = "ALIVE"
// 		logger.TestLog("Liveness", sm)
// 		time.Sleep(time.Duration(interval) * time.Second)
// 	}
// }
// func mock_telelmetry(interval int) {
// 	for true {
// 		sm := new(l.SubValueType)
// 		sm.AssetInfo.AssetId = "Welcome"
// 		sm.AssetInfo.Size = 500 + rand.Float64()*(1500-500)
// 		sm.AssetInfo.RelativeLocation = "EROS/Welcome"
// 		logger.TestLog("AssetDownloadOnDeviceFromSES", sm)
// 		logger.TestLog("AssetDeleteOnDeviceByScheduler", sm)
// 		logger.TestLog("AssetDeleteOnDeviceByCommand", sm)
// 		logger.TestLog("FailedAssetDeleteOnDeviceByScheduler", sm)
// 		logger.TestLog("FailedAssetDeleteOnDeviceByCommand", sm)
// 		sm1 := new(l.SubValueType)
// 		sm1.AssetInfo.AssetId = "Welcome"
// 		sm1.AssetInfo.Size = 500 + rand.Float64()*(1500-500)
// 		sm1.AssetInfo.RelativeLocation = "EROS/Welcome"
// 		curr := time.Now()
// 		sm1.AssetInfo.StartTime = curr.Unix()
// 		d := rand.Intn(300)
// 		sm1.AssetInfo.Duration = d
// 		endTime := curr.Add(time.Duration(d) * time.Second)
// 		sm1.AssetInfo.EndTime = endTime.Unix()
// 		sm1.AssetInfo.IsUpdate = false
// 		logger.TestLog("AssetDownloadOnDeviceByCommand", sm1)
// 		sm2 := new(l.SubValueType)
// 		sm2.AssetInfo.AssetId = "Welcome"
// 		sm2.AssetInfo.Size = 600
// 		logger.TestLog("FailedAssetDownloadOnMobile", sm2)
// 		curr = time.Now()
// 		sm2.AssetInfo.StartTime = curr.Unix()
// 		d = rand.Intn(300)
// 		sm2.AssetInfo.Duration = d
// 		endTime = curr.Add(time.Duration(d) * time.Second)
// 		sm2.AssetInfo.EndTime = endTime.Unix()
// 		//fmt.Println(sm2)
// 		logger.TestLog("SucessfulAssetDownloadOnMobile", sm2)
// 		im := new(l.SubValueType)
// 		im.IntegrityStats.AssetId = "Welcome"
// 		im.IntegrityStats.RelativeLocation = "EROS/Welcome"
// 		im.IntegrityStats.Filename = "vod.mp4"
// 		im.IntegrityStats.ActualSHA = "qwwerrrrrrr"
// 		im.IntegrityStats.ExpectedSHA = "asdffgggf"
// 		logger.TestLog("CorruptedFileStatsFromScheduler", im)
// 		time.Sleep(time.Duration(interval) * time.Second)
// 	}
// }
func mock_hubstorageandmemory(interval int) {
	for true {
		sm := new(l.MessageSubType)
		sm.FloatValue = 20 + rand.Float64()*(500-20)
		logger.Log("HubStorage", sm)
		//sm.FloatValue = 40 + rand.Float64()*(100-40)
		//logger.TestLog("Memory", sm)
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
