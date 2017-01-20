package thingdoer_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/cloudfoundry-incubator/diego-enabler/api"
	"github.com/cloudfoundry-incubator/diego-enabler/api/apifakes"
	"github.com/cloudfoundry-incubator/diego-enabler/models"
	"github.com/cloudfoundry-incubator/diego-enabler/thingdoer"
	"github.com/cloudfoundry-incubator/diego-enabler/thingdoer/thingdoerfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeaApps", func() {
	var (
		fakePaginatedRequester    *thingdoerfakes.FakePaginatedRequester
		fakeApplicationsParser    *thingdoerfakes.FakeApplicationsParser
		fakeCloudControllerClient *apifakes.FakeCloudControllerClient
		apps                      models.Applications

		command thingdoer.AppsGetter
		err     error
	)

	BeforeEach(func() {
		fakePaginatedRequester = new(thingdoerfakes.FakePaginatedRequester)
		fakeCloudControllerClient = new(apifakes.FakeCloudControllerClient)
		fakePaginatedRequester.HttpClientReturns(fakeCloudControllerClient)
		fakeApplicationsParser = new(thingdoerfakes.FakeApplicationsParser)
		command = thingdoer.AppsGetter{}
	})

	JustBeforeEach(func() {
		client := &api.Client{
			BaseUrl: new(url.URL),
		}
		apps, err = command.DeaApps(fakeApplicationsParser, fakePaginatedRequester, client)
	})

	It("should create a request with diego filter set to false", func() {
		expectedFilters := api.Filters{
			api.EqualFilter{
				Name:  "diego",
				Value: false,
			},
		}

		Expect(fakePaginatedRequester.DoCallCount()).To(Equal(1))
		filters, _ := fakePaginatedRequester.DoArgsForCall(0)
		Expect(filters).To(Equal(expectedFilters))
	})

	Context("when an organization name is specified", func() {
		BeforeEach(func() {
			command.OrganizationGuid = "some-organization-guid"
		})

		It("should create a request with organization guid set", func() {
			expectedFilters := api.Filters{
				api.EqualFilter{
					Name:  "diego",
					Value: false,
				},
				api.EqualFilter{
					Name:  "organization_guid",
					Value: "some-organization-guid",
				},
			}

			Expect(fakePaginatedRequester.DoCallCount()).To(Equal(1))
			filters, _ := fakePaginatedRequester.DoArgsForCall(0)
			Expect(filters).To(Equal(expectedFilters))
		})
	})

	Context("when an space name is specified", func() {
		BeforeEach(func() {
			command.SpaceGuid = "some-space-guid"
		})

		It("should create a request with space guid set", func() {
			expectedFilters := api.Filters{
				api.EqualFilter{
					Name:  "diego",
					Value: false,
				},
				api.EqualFilter{
					Name:  "space_guid",
					Value: "some-space-guid",
				},
			}

			Expect(fakePaginatedRequester.DoCallCount()).To(Equal(1))
			filters, _ := fakePaginatedRequester.DoArgsForCall(0)
			Expect(filters).To(Equal(expectedFilters))
		})
	})

	Context("when the paginated requester fails", func() {
		var requestError error

		BeforeEach(func() {
			requestError = errors.New("making API requests failed")
			fakePaginatedRequester.DoReturns([][]byte{}, requestError)
		})

		It("returns the requester error", func() {
			Expect(apps).To(BeEmpty())
			Expect(err).To(Equal(requestError))
		})
	})

	Context("When the paginated requester succeeds", func() {
		BeforeEach(func() {
			responseBodies := [][]byte{
				[]byte("some-json"),
				[]byte("some-other-json"),
			}
			fakePaginatedRequester.DoReturns(responseBodies, nil)
		})

		Context("when the parsing fails", func() {
			var apps models.Applications
			var parseError error

			BeforeEach(func() {
				parseError = errors.New("parsing json failed")
				fakeApplicationsParser.ParseReturns(apps, parseError)
			})

			It("returns the parse error", func() {
				Expect(apps).To(BeEmpty())
				Expect(err).To(Equal(parseError))
			})
		})

		Context("when the parsing succeeds", func() {
			var parsedApps models.Applications = models.Applications{
				models.Application{
					models.ApplicationEntity{
						Diego: false,
					},
					models.ApplicationMetadata{
						Guid: "some-guid",
					},
				},
			}

			BeforeEach(func() {
				// for each call of Parse
				fakeApplicationsParser.ParseReturns(parsedApps, nil)
				fakeCloudControllerClient.DoStub = func(*http.Request) (*http.Response, error) {
					return &http.Response{Body: ioutil.NopCloser(strings.NewReader(`{"total_results": 5}`))}, nil
				}
			})

			It("returns a list of diego applications", func() {
				expectedApps := models.Applications{
					models.Application{
						models.ApplicationEntity{
							Diego:          false,
							NumberOfRoutes: 5,
						},
						models.ApplicationMetadata{
							Guid: "some-guid",
						},
					},
					models.Application{
						models.ApplicationEntity{
							Diego:          false,
							NumberOfRoutes: 5,
						},
						models.ApplicationMetadata{
							Guid: "some-guid",
						},
					},
				}

				Expect(err).NotTo(HaveOccurred())
				Expect(apps).To(Equal(expectedApps))
			})

			PContext("when getting routes for an application fails", func() {
			})
		})
	})
})
