package server

import (
	"bytes"
	"github.com/mysterium/node/requestor"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"testing"
)

var testRequestApiUrl = "http://testUrl"

func TestHttpErrorIsReportedAsErrorReturnValue(t *testing.T) {
	req, err := requestor.NewGetRequest(testRequestApiUrl, "path", nil)
	assert.NoError(t, err)

	response := &http.Response{
		StatusCode: 400,
		Request:    req,
	}
	err = parseResponseError(response)
	assert.Error(t, err)
}

type testResponse struct {
	MyValue string `json:"myValue"`
}

func TestHttpResponseBodyIsParsedCorrectly(t *testing.T) {

	req, err := requestor.NewGetRequest(testRequestApiUrl, "path", nil)
	assert.NoError(t, err)

	response := &http.Response{
		StatusCode: 200,
		Request:    req,
		Body: ioutil.NopCloser(bytes.NewBufferString(
			`{
				"myValue" : "abc"
			}`)),
	}
	var testDto testResponse
	err = parseResponseJson(response, &testDto)
	assert.NoError(t, err)
	assert.Equal(t, testResponse{"abc"}, testDto)

}
