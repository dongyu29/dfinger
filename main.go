package main

import (
	"dfinger/common"
	"dfinger/core/finger"
	"dfinger/core/network"
)

func main() {
	common.Dfinger_init()
	var err error
	file := common.Infos.FingerFile
	finger.Rules, err = finger.LoadFingerprints(file)
	if err != nil {
		panic(err)
	}
	//覆写
	common.ParseInfo.UrlInfos = finger.GenerateWebscanTasks(common.ParseInfo.Iplist, common.ParseInfo.Portlist)

	finger.Run(common.ParseInfo.UrlInfos, network.NewDefaultHTTPClient())
}
