package recipe

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/JulzDiverse/cfclient"
	"github.com/pkg/errors"
)

type DropletUploader struct {
	Config *cfclient.Config
}

func (u *DropletUploader) Upload(guid string, path string) error {
	if guid == "" {
		return errors.New("empty guid parameter")
	}

	if path == "" {
		return errors.New("empty path parameter")
	}

	uploadURI := fmt.Sprintf("https://cloud-controller-ng.service.cf.internal:9023/internal/v4/droplets/%s/upload", guid)
	endpoint := fmt.Sprintf("https://cc-uploader.service.cf.internal:9091/v1/droplet/%s?cc-droplet-upload-uri=%s", guid, uploadURI)
	// return u.pushDroplet(path, guid, endpoint)
	return u.UploadDeDroplette(path, endpoint)
}

func (u *DropletUploader) UploadCache(appGUID, stagingGUID, path string) error {
	if path == "" {
		return errors.New("empty path parameter")
	}
	stack := "cflinuxfs2"

	uploadURI := fmt.Sprintf("https://cloud-controller-ng.service.cf.internal:9023/internal/v4/buildpack_cache/%s/%s/upload", stack, appGUID)
	endpoint := fmt.Sprintf("https://cc-uploader.service.cf.internal:9091/v1/build_artifacts/%s?cc-build-artifacts-upload-uri=%s", stagingGUID, uploadURI)

	return u.pushDroplet(path, "whatever", endpoint)
}

///////////////////////////////

func (u *DropletUploader) UploadDeDroplette(path, url string) error {
	file, size, md5, err := u.prepareFileForUpload(path)
	if err != nil {
		fmt.Println("FAILED THE PREPARATION")
		return err
	}

	return u.attemptUpload(file, size, md5, url)
}

func (u *DropletUploader) prepareFileForUpload(fileLocation string) (*os.File, int64, string, error) {
	sourceFile, err := os.Open(fileLocation)
	if err != nil {
		return nil, 0, "", err
	}

	fileInfo, err := sourceFile.Stat()
	if err != nil {
		return nil, 0, "", err
	}

	contentHash := md5.New()
	_, err = io.Copy(contentHash, sourceFile)
	if err != nil {
		return nil, 0, "", err
	}

	contentMD5 := base64.StdEncoding.EncodeToString(contentHash.Sum(nil))

	return sourceFile, fileInfo.Size(), contentMD5, nil
}

func (u *DropletUploader) attemptUpload(
	sourceFile *os.File,
	bytesToUpload int64,
	contentMD5 string,
	url string) error {

	_, err := sourceFile.Seek(0, 0)
	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", url, ioutil.NopCloser(sourceFile))
	if err != nil {
		return err
	}

	request.ContentLength = bytesToUpload
	request.Header.Set("Content-Type", "application/octet-stream")
	request.Header.Set("Content-MD5", contentMD5)

	resp, err := u.Config.HttpClient.Do(request)
	if err != nil {
		return err
	}
	fmt.Println("REPONMSASDASDA IS : ", resp)

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Upload failed: Status code %d", resp.StatusCode)
	}

	return nil
}

//////////////

func (u *DropletUploader) pushDroplet(path string, guid, endpoint string) error {
	name := filepath.Base(path)

	droplet, size, err := u.readFile(path)
	if err != nil {
		return err
	}
	defer droplet.Close()
	return u.setDroplet(name, guid, droplet, size, endpoint)
}

func (u *DropletUploader) readFile(path string) (io.ReadCloser, int64, error) {
	return u.openFile(path, os.O_RDONLY, 0)
}

func (u *DropletUploader) openFile(path string, flag int, perm os.FileMode) (*os.File, int64, error) {
	file, err := os.OpenFile(path, flag, perm)
	if err != nil {
		return nil, 0, err
	}
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, 0, err
	}
	return file, fileInfo.Size(), nil
}

func (u *DropletUploader) setDroplet(filename, guid string, droplet io.Reader, size int64, endpoint string) error {
	// fieldname := "droplet"
	// extension := filepath.Ext(filename)
	// name := filename[0 : len(filename)-len(extension)]

	// // This is necessary because (similar to S3) CC does not accept chunked multipart MIME
	// contentLength := emptyMultipartSize(fieldname, filename) + size

	// readBody, writeBody := io.Pipe()
	// defer readBody.Close()

	// form := multipart.NewWriter(writeBody)
	// errChan := make(chan error, 1)
	// go func() {
	// 	defer writeBody.Close()

	// 	dropletPart, err := form.CreateFormFile(fieldname, filename)
	// 	if err != nil {
	// 		errChan <- err
	// 		return
	// 	}
	// 	if _, err := io.CopyN(dropletPart, droplet, size); err != nil {
	// 		errChan <- err
	// 		return
	// 	}
	// 	errChan <- form.Close()
	// }()

	if err := u.putJob("", guid, endpoint, droplet, "application/gzip", 0); err != nil {
		// <-errChan
		return err
	}

	return nil
	// return <-errChan
}

func emptyMultipartSize(fieldname, filename string) int64 {
	body := &bytes.Buffer{}
	form := multipart.NewWriter(body)
	form.CreateFormFile(fieldname, filename)
	form.Close()
	return int64(body.Len())
}

func (u *DropletUploader) putJob(name, guid, endpoint string, body io.Reader, contentType string, contentLength int64) error {
	response, err := u.doAppRequest(name, guid, "PUT", endpoint, body, contentType, contentLength, http.StatusCreated)
	if err != nil {
		return err
	}
	response.Body.Close()
	return nil
}

func (u *DropletUploader) doAppRequest(name, guid, method, endpoint string, body io.Reader, contentType string, contentLength int64, desiredStatus int) (*http.Response, error) {
	response, err := u.doRequest("POST", endpoint, body, contentType, contentLength, desiredStatus)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (u *DropletUploader) doRequest(method, endpoint string, body io.Reader, contentType string, contentLength int64, desiredStatus int) (*http.Response, error) {
	//targetURL.Path = path.Join("https://api.bosh-lite-cube.dynamic-dns.net", endpoint)
	r := u.NewRequestWithBody(method, endpoint, body)
	request, err := r.toHTTP()
	if err != nil {
		return nil, err
	}

	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	if contentLength > 0 {
		request.ContentLength = contentLength
	}

	response, err := u.DoHttpRequest(request)
	if err != nil {
		return nil, err
	}
	// if response.StatusCode != desiredStatus {
	// 	response.Body.Close()return
	// 	return nil, fmt.Errorf("unexpected '%s' from: %s %s", response.Status, method, endpoint)
	// }
	return response, nil
}

type request struct {
	method string
	url    string
	params url.Values
	body   io.Reader
	obj    interface{}
}

// NewRequestWithBody e zle
func (u *DropletUploader) NewRequestWithBody(method, path string, body io.Reader) *request {
	r := u.NewRequest(method, path)

	// Set request body
	r.body = body

	return r
}

func (u *DropletUploader) NewRequest(method, path string) *request {
	r := &request{
		method: method,
		url:    path,
		params: make(map[string][]string),
	}
	return r
}
func (u *DropletUploader) DoHttpRequest(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", "Go-CF-client/1.1")
	resp, err := u.Config.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		var cfErr CloudFoundryError
		if err := decodeBody(resp, &cfErr); err != nil {
			return resp, errors.Wrap(err, "Unable to decode body")
		}
		return nil, cfErr
	}
	return resp, nil
}

// decodeBody is used to JSON decode a body
func decodeBody(resp *http.Response, out interface{}) error {
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(out)
}

// toHTTP converts the request to an HTTP request
func (r *request) toHTTP() (*http.Request, error) {

	// Check if we should encode the body
	if r.body == nil && r.obj != nil {
		b, err := encodeBody(r.obj)
		if err != nil {
			return nil, err
		}
		r.body = b
	}

	// Create the HTTP request
	return http.NewRequest(r.method, r.url, r.body)
}

// encodeBody is used to encode a request body
func encodeBody(obj interface{}) (io.Reader, error) {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	if err := enc.Encode(obj); err != nil {
		return nil, err
	}
	return buf, nil
}

// ERORORORROR
type CloudFoundryErrors struct {
	Errors []CloudFoundryError `json:"errors"`
}

func (cfErrs CloudFoundryErrors) Error() string {
	err := ""

	for _, cfErr := range cfErrs.Errors {
		err += fmt.Sprintf("%s\n", cfErr)
	}

	return err
}

type CloudFoundryError struct {
	Code        int    `json:"code"`
	ErrorCode   string `json:"error_code"`
	Description string `json:"description"`
}

func (cfErr CloudFoundryError) Error() string {
	return fmt.Sprintf("cfclient: error (%d): %s", cfErr.Code, cfErr.ErrorCode)
}
