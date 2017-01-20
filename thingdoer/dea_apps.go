package thingdoer

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/cloudfoundry-incubator/diego-enabler/api"
	"github.com/cloudfoundry-incubator/diego-enabler/models"
)

// ccResponse is used only to unmarshal the top level properties of a CC
// response.
type ccResponse struct {
	TotalResults int `json:"total_results"`
}

func (c AppsGetter) DeaApps(appsParser ApplicationsParser, paginatedRequester PaginatedRequester, client *api.Client) (models.Applications, error) {
	var noApps models.Applications

	filter := api.Filters{
		api.EqualFilter{
			Name:  "diego",
			Value: false,
		},
	}

	if c.OrganizationGuid != "" {
		filter = append(
			filter,
			api.EqualFilter{
				Name:  "organization_guid",
				Value: c.OrganizationGuid,
			},
		)
	} else if c.SpaceGuid != "" {
		filter = append(
			filter,
			api.EqualFilter{
				Name:  "space_guid",
				Value: c.SpaceGuid,
			},
		)
	}

	params := map[string]interface{}{}

	responseBodies, err := paginatedRequester.Do(filter, params)
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

	// for each application, a request is made to the application's route url to
	// get it's number of routes.
	for i, app := range applications {
		routeReq := &http.Request{
			Method: "GET",
			URL:    client.BaseUrl,
		}
		routeReq.URL.Path = app.RoutesURL
		routeReq.URL.RawQuery = ""
		routeReq.Header = http.Header{
			"Authorization": {client.AuthToken},
		}

		httpClient := paginatedRequester.HttpClient()

		// the code does not specifically handle a non 200 status code.
		// in the case of a non 200 status code, the NumberOfRoutes will default to 0
		routeRes, err := httpClient.Do(routeReq)
		if err != nil {
			return noApps, err
		}

		rawBody, err := ioutil.ReadAll(routeRes.Body)
		if err != nil {
			return noApps, err
		}

		var unmarshaledBody ccResponse
		err = json.Unmarshal(rawBody, &unmarshaledBody)
		if err != nil {
			return noApps, err
		}

		applications[i].NumberOfRoutes = unmarshaledBody.TotalResults
	}

	return applications, nil
}
