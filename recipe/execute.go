package recipe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/buildpackapplifecycle"
	"code.cloudfoundry.org/runtimeschema/cc_messages"
	"github.com/pkg/errors"
)

const workspaceDir = "/workspace"

type IOCommander struct {
	Stdout *os.File
	Stderr *os.File
	Stdin  *os.File
}

func (c *IOCommander) Exec(cmd string, args ...string) error {
	command := exec.Command(cmd, args...)
	command.Stdout = c.Stdout
	command.Stderr = c.Stderr
	command.Stdin = c.Stdin

	return command.Run()
}

type PacksBuilderConf struct {
	BuildpacksDir             string
	OutputDropletLocation     string
	OutputBuildArtifactsCache string
	OutputMetadataLocation    string
}

type PacksExecutor struct {
	Conf      PacksBuilderConf
	Installer Installer
	Uploader  Uploader
	Commander Commander
}

func (e *PacksExecutor) ExecuteRecipe(appID, stagingGUID, completionCallback, eiriniAddr, providedBuildpacksJSON string) error {
	err := e.Installer.Install(appID, workspaceDir)
	if err != nil {
		respondWithFailure(err, stagingGUID, completionCallback, eiriniAddr)
		return err
	}

	args := []string{
		"-buildpacksDir", e.Conf.BuildpacksDir,
		"-outputDroplet", e.Conf.OutputDropletLocation,
		"-outputBuildArtifactsCache", e.Conf.OutputBuildArtifactsCache,
		"-outputMetadata", e.Conf.OutputMetadataLocation,
	}

	err = e.Commander.Exec("/packs/builder", args...)
	if err != nil {
		respondWithFailure(err, stagingGUID, completionCallback, eiriniAddr)
		return err
	}

	fmt.Println("Start Upload Process.")
	err = e.Uploader.Upload(stagingGUID, e.Conf.OutputDropletLocation)
	if err != nil {
		respondWithFailure(err, stagingGUID, completionCallback, eiriniAddr)
		return err
	}
	// fmt.Printf("Staging GUID is %s. App ID is %s. Starting Upload for build artifacts", stagingGUID, appID)
	// err = e.Uploader.UploadCache(appID, stagingGUID, e.Conf.OutputBuildArtifactsCache)
	// if err != nil {
	// 	fmt.Println("FAILED BUILD ARTIFACTS UPLOADS: ", err)
	// 	respondWithFailure(err, stagingGUID, completionCallback, eiriniAddr)
	// 	return err
	// }

	fmt.Println("Completion Callback is: ", completionCallback)

	cbResponse := e.createSuccessResponse(stagingGUID, completionCallback, providedBuildpacksJSON)
	return sendCompleteResponse(eiriniAddr, cbResponse)
}

func (e *PacksExecutor) createSuccessResponse(stagingGUID, completionCallback, providedBuildpacksJSON string) *models.TaskCallbackResponse {
	providedBuildpacks, _ := getProvidedBuildpacks(providedBuildpacksJSON)
	stagingResult, _ := getStagingResult(e.Conf.OutputMetadataLocation)
	stagingResult, _ = modifyBuildpackKey(stagingResult, providedBuildpacks)

	result, err := json.Marshal(stagingResult)
	if err != nil {
		panic(err)
	}

	annotation := cc_messages.StagingTaskAnnotation{
		CompletionCallback: completionCallback,
	}

	annotationJSON, _ := json.Marshal(annotation)

	fmt.Println("STAGING RESULT AFTER ALL THE SHENANIGANS IS: ", stagingResult)

	return &models.TaskCallbackResponse{
		TaskGuid:   stagingGUID,
		Result:     string(result),
		Failed:     false,
		Annotation: string(annotationJSON),
	}
}

func createFailureResponse(failure error, stagingGUID, completionCallback string) *models.TaskCallbackResponse {
	annotation := cc_messages.StagingTaskAnnotation{
		CompletionCallback: completionCallback,
	}

	annotationJSON, err := json.Marshal(annotation)
	if err != nil {
		panic(err)
	}

	return &models.TaskCallbackResponse{
		TaskGuid:      stagingGUID,
		Failed:        true,
		FailureReason: failure.Error(),
		Annotation:    string(annotationJSON),
	}
}

func respondWithFailure(failure error, stagingGUID, completionCallback, eiriniAddr string) {
	cbResponse := createFailureResponse(failure, stagingGUID, completionCallback)

	if completeErr := sendCompleteResponse(eiriniAddr, cbResponse); completeErr != nil {
		fmt.Println("Error processsing completion callback:", completeErr.Error())
	}
}

func getProvidedBuildpacks(buildpacksJSON string) ([]cc_messages.Buildpack, error) {
	var providedBuildpacks []cc_messages.Buildpack
	err := json.Unmarshal([]byte(buildpacksJSON), &providedBuildpacks)
	if err != nil {
		return []cc_messages.Buildpack{}, err
	}

	return providedBuildpacks, nil
}

func modifyBuildpackKey(stagingResult buildpackapplifecycle.StagingResult, providedBuildpacks []cc_messages.Buildpack) (buildpackapplifecycle.StagingResult, error) {
	buildpackName := stagingResult.LifecycleMetadata.BuildpackKey
	buildpackKey, _ := getBuildpackKey(buildpackName, providedBuildpacks)

	stagingResult.LifecycleMetadata.BuildpackKey = buildpackKey
	stagingResult.LifecycleMetadata.Buildpacks[0].Key = buildpackKey

	return stagingResult, nil
}

func getBuildpackKey(name string, providedBuildpacks []cc_messages.Buildpack) (string, error) {
	for _, b := range providedBuildpacks {
		if b.Name == name {
			return b.Key, nil
		}
	}

	return "", fmt.Errorf("could not find buildpack with name: %s", name)
}

func getStagingResult(path string) (buildpackapplifecycle.StagingResult, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return buildpackapplifecycle.StagingResult{}, errors.Wrap(err, "failed to read result.json")
	}
	var stagingResult buildpackapplifecycle.StagingResult
	err = json.Unmarshal(contents, &stagingResult)
	if err != nil {
		return buildpackapplifecycle.StagingResult{}, err
	}
	return stagingResult, nil
}

func readResultJSON(path string) ([]byte, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return []byte{}, errors.Wrap(err, "failed to read result.json")
	}
	return file, nil
}

func sendCompleteResponse(eiriniAddress string, response *models.TaskCallbackResponse) error {
	responseJSON, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}

	uri := fmt.Sprintf("http://%s/stage/%s/completed", eiriniAddress, response.TaskGuid)
	// uri := fmt.Sprintf("%s/stage/%s/completed", eiriniAddress, response.TaskGuid)
	req, err := http.NewRequest("PUT", uri, bytes.NewBuffer(responseJSON))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "request failed")
	}

	fmt.Println("RESPONSE IS: ", resp)
	if resp.StatusCode >= 400 {
		return errors.New("Request not successful")
	}

	return nil
}
