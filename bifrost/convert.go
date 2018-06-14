package bifrost

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/eirini"
	"code.cloudfoundry.org/eirini/opi"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/runtimeschema/cc_messages"
)

func Convert(
	msg cc_messages.DesireAppRequestFromCC,
	registryUrl string,
	registryIP string,
	cfClient eirini.CfClient,
	client *http.Client,
	log lager.Logger,
) opi.LRP {
	if msg.DockerImageUrl == "" {
		msg.DockerImageUrl = dropletToImageURI(msg, cfClient, client, registryUrl, registryIP, log)
	}

	envMap := envVarsToMap(msg.Environment)
	vcap := parseVcapApplication(envMap)

	metadata := map[string]string{}
	metadata["process_guid"] = msg.ProcessGuid
	metadata = merge(metadata, vcap)

	return opi.LRP{
		Name:            vcap["application_id"],
		Image:           msg.DockerImageUrl,
		TargetInstances: msg.NumInstances,
		Command: []string{
			msg.StartCommand,
		},
		Env:      envMap,
		Metadata: metadata,
	}
}

func parseVcapApplication(env map[string]string) map[string]string {
	metadata := map[string]string{}
	if vcap, ok := env["VCAP_APPLICATION"]; ok {
		if err := json.Unmarshal([]byte(vcap), &metadata); err != nil {
			panic(err)
		}
	}
	return metadata
}

func envVarsToMap(envs []*models.EnvironmentVariable) map[string]string {
	envMap := map[string]string{}
	for _, v := range envs {
		envMap[v.Name] = v.Value
	}
	return envMap
}

func merge(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

func dropletToImageURI(
	msg cc_messages.DesireAppRequestFromCC,
	cfClient eirini.CfClient,
	client *http.Client,
	registryUrl string,
	registryIP string,
	log lager.Logger,
) string {

	// we can use the map now instead of appInfo
	var appInfo eirini.AppInfo
	for _, v := range msg.Environment {
		if v.Name == "VCAP_APPLICATION" {
			err := json.Unmarshal([]byte(v.Value), &appInfo)
			if err != nil {
				log.Error("failed-to-decode-environment-json-from-cc_message", err)
				panic(err)
			}
		}
	}

	dropletBytes, err := cfClient.GetDropletByAppGuid(appInfo.AppGuid)
	if err != nil {
		log.Error("failed-to-get-droplet-from-cloud-controller", err, lager.Data{"app-guid": appInfo.AppGuid})
		panic(err)
	}

	stageRequest(client, registryUrl, appInfo, msg.DropletHash, dropletBytes, log)

	return fmt.Sprintf("%s/cloudfoundry/app-name:%s", registryIP, msg.DropletHash)
}

func stageRequest(
	client *http.Client,
	registryUrl string,
	appInfo eirini.AppInfo,
	dropletHash string,
	dropletBytes []byte,
	log lager.Logger,
) string {
	registryStageUri := registryStageUri(registryUrl, appInfo.SpaceName, appInfo.AppName, dropletHash)

	log.Info("sending-request-to-registry", lager.Data{"request": registryStageUri})

	req, err := http.NewRequest("POST", registryStageUri, bytes.NewReader(dropletBytes))
	if err != nil {
		log.Error("failed-to-create-http-request", err, nil)
		panic(err)
	}

	req.Header.Set("Content-Type", "application/gzip")

	resp, err := client.Do(req)
	if err != nil {
		log.Error("stage-request-to-registry-failed", err, lager.Data{"request": registryStageUri})
		return ""
	}

	log.Info("request-successful", lager.Data{"response_status": resp.StatusCode})

	digest, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("read-response-failed", err)
		return ""
	}

	return string(digest)

}

func dropletDownloadUri(baseUrl string, appGuid string) string {
	return fmt.Sprintf("%s/v2/apps/%s/droplet/download", baseUrl, appGuid)
}

func registryStageUri(baseUrl string, space string, appname string, guid string) string {
	return fmt.Sprintf("%s/v2/%s/%s/blobs/?guid=%s", baseUrl, space, appname, guid)
}
