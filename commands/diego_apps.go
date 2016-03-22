package commands

import (
	"net/http"

	"io/ioutil"

	"github.com/cloudfoundry-incubator/diego-enabler/api"
	"github.com/cloudfoundry-incubator/diego-enabler/models"
)

type RequestFactory func(api.Filter, map[string]interface{}) (*http.Request, error)

//go:generate counterfeiter . CloudControllerClient
type CloudControllerClient interface {
	Do(*http.Request) (*http.Response, error)
}

//go:generate counterfeiter . ApplicationsParser
type ApplicationsParser interface {
	Parse([]byte) (models.Applications, error)
}

//go:generate counterfeiter . PaginatedParser
type PaginatedParser interface {
	Parse([]byte) (api.PaginatedResponse, error)
}

func DiegoApps(requestFactory RequestFactory, client CloudControllerClient, appsParser ApplicationsParser, pageParser PaginatedParser) (models.Applications, error) {
	var noApps models.Applications

	filter := api.EqualFilter{
		Name:  "diego",
		Value: true,
	}

	params := map[string]interface{}{}

	responseBodies, err := paginatedRequester(requestFactory, filter, params, client, pageParser)
	if err != nil {
		return noApps, err
	}

	var applications models.Applications

	for _, nextBody := range responseBodies {
		apps, err := appsParser.Parse(nextBody)
		if err != nil {
			return noApps, err
		}

		applications = append(applications, apps...)
	}

	return applications, nil
}

func paginatedRequester(requestFactory RequestFactory, filter api.Filter, params map[string]interface{}, client CloudControllerClient, pageParser PaginatedParser) ([][]byte, error) {
	var noBodies [][]byte

	req, err := requestFactory(filter, params)
	if err != nil {
		return noBodies, err
	}

	var responseBodies [][]byte

	res, err := client.Do(req)
	if err != nil {
		return noBodies, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return noBodies, err
	}

	responseBodies = append(responseBodies, body)

	paginatedRes, err := pageParser.Parse(body)
	if err != nil {
		return noBodies, err
	}
	for page := 2; page <= paginatedRes.TotalPages; page++ {
		// construct a new request with the current page
		params["page"] = page
		req, err := requestFactory(filter, params)
		if err != nil {
			return noBodies, err
		}

		// perform the request
		res, err := client.Do(req)
		if err != nil {
			return noBodies, err
		}

		body, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return noBodies, err
		}

		responseBodies = append(responseBodies, body)
	}

	return responseBodies, nil
}
