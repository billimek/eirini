package handler

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/eirini"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/runtimeschema/cc_messages"
	"github.com/julienschmidt/httprouter"
)

type Stage struct {
	stager eirini.Stager
	logger lager.Logger
}

func NewStageHandler(stager eirini.Stager, logger lager.Logger) *Stage {
	logger = logger.Session("staging-handler")

	return &Stage{
		stager: stager,
		logger: logger,
	}
}

func (s *Stage) Stage(resp http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	stagingGUID := ps.ByName("staging_guid")
	logger := s.logger.Session("staging-request", lager.Data{"staging-guid": stagingGUID})

	requestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Error("read-body-failed", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	var stagingRequest cc_messages.StagingRequestFromCC
	err = json.Unmarshal(requestBody, &stagingRequest)
	if err != nil {
		logger.Error("unmarshal-request-failed", err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	envVars := []string{}
	for _, envVar := range stagingRequest.Environment {
		envVars = append(envVars, envVar.Name)
	}

	logger.Info("environment", lager.Data{"keys": envVars})

	err = s.stager.DesireTask(stagingGUID, stagingRequest)
	if err != nil {
		logger.Error("stage-app-failed", err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
}

func (s *Stage) StagingComplete(res http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	stagingGUID := ps.ByName("staging_guid")
	logger := s.logger.Session("staging-complete", lager.Data{"staging-guid": stagingGUID})

	task := &models.TaskCallbackResponse{}
	err := json.NewDecoder(req.Body).Decode(task)
	if err != nil {
		logger.Error("parsing-incoming-task-failed", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	var annotation cc_messages.StagingTaskAnnotation
	err = json.Unmarshal([]byte(task.Annotation), &annotation)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		logger.Error("parsing-annotation-failed", err)
		return
	}

	response, err := s.stager.BuildStagingResponse(task)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		logger.Error("error-creating-staging-response", err)
		return
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		logger.Error("get-staging-response-failed", err)
		return
	}

	request, err := http.NewRequest("POST", annotation.CompletionCallback, bytes.NewBuffer(responseJSON))
	if err != nil {
		return
	}

	client := http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		logger.Error("cc-staging-complete-failed", err)
		return
	}

	logger.Info("staging-complete-request-finished-with-status", lager.Data{"StatusCode": resp.StatusCode})
	logger.Info("posted-staging-complete")
}
