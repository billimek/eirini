package bifrost

import (
	"fmt"
	"net/http"

	"code.cloudfoundry.org/eirini"
	"code.cloudfoundry.org/eirini/opi"
	"code.cloudfoundry.org/lager"
)

type VcapApp struct {
	AppName   string   `json:"application_name"`
	AppId     string   `json:"application_id"`
	Version   string   `json:"version"`
	AppUris   []string `json:"application_uris"`
	SpaceName string   `json:"space_name"`
}

type Converter interface {
	Convert(request eirini.DesireLRPRequest, registryUrl string, registryIP string, cfClient eirini.CfClient, client *http.Client, log lager.Logger) opi.LRP
}

type ConvertFunc func(request eirini.DesireLRPRequest, registryUrl string, registryIP string, cfClient eirini.CfClient, client *http.Client, log lager.Logger) opi.LRP

func (fn ConvertFunc) Convert(request eirini.DesireLRPRequest, registryUrl string, registryIP string, cfClient eirini.CfClient, client *http.Client, log lager.Logger) opi.LRP {
	return fn(request, registryUrl, registryIP, cfClient, client, log)
}

func dropletDownloadUri(baseUrl string, appGuid string) string {
	return fmt.Sprintf("%s/v2/apps/%s/droplet/download", baseUrl, appGuid)
}

func registryStageUri(baseUrl string, space string, appname string, guid string) string {
	return fmt.Sprintf("%s/v2/%s/%s/blobs/?guid=%s", baseUrl, space, appname, guid)
}
