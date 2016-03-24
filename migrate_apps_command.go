package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cloudfoundry-incubator/diego-enabler/api"
	"github.com/cloudfoundry-incubator/diego-enabler/diego_support"
	"github.com/cloudfoundry-incubator/diego-enabler/models"
	"github.com/cloudfoundry-incubator/diego-enabler/thingdoer"
	"github.com/cloudfoundry-incubator/diego-enabler/ui"
	"github.com/cloudfoundry/cli/plugin"
	"github.com/jessevdk/go-flags"
)

type MigrateApps struct {
	opts               Opts
	runtime            ui.Runtime
	appsGetterFunc     thingdoer.AppsGetterFunc
	migrateAppsCommand *ui.MigrateAppsCommand
}

func PrepareMigrateApps(args []string, cliConnection plugin.CliConnection) (MigrateApps, error) {
	empty := MigrateApps{}
	var opts Opts
	_, err := flags.ParseArgs(&opts, args)
	if err != nil {
		return empty, err
	}
	runtime, err := ui.ParseRuntime(args[0])
	if err != nil {

		return empty, err
	}

	appsGetter, err := NewAppsGetterFunc(cliConnection, opts.Organization, runtime.Flip())
	if err != nil {
		return empty, err
	}

	migrateAppsCommand, err := newMigrateAppsCommand(cliConnection, opts, runtime)
	if err != nil {
		return empty, err
	}

	return MigrateApps{
		opts:               opts,
		runtime:            runtime,
		appsGetterFunc:     appsGetter,
		migrateAppsCommand: &migrateAppsCommand,
	}, nil
}

func (cmd *MigrateApps) Execute(cliConnection plugin.CliConnection) error {
	cmd.migrateAppsCommand.BeforeAll()

	if err := verifyLoggedIn(cliConnection); err != nil {
		return err
	}

	accessToken, err := cliConnection.AccessToken()
	if err != nil {
		return err
	}

	pageParser := api.PageParser{}
	appsParser := models.ApplicationsParser{}
	spacesParser := models.SpacesParser{}

	apiEndpoint, err := cliConnection.ApiEndpoint()
	if err != nil {
		return err
	}

	apiClient, err := api.NewApiClient(apiEndpoint, accessToken)
	if err != nil {
		return err
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	appRequestFactory := apiClient.HandleFiltersAndParameters(
		apiClient.Authorize(apiClient.NewGetAppsRequest),
	)

	apps, err := cmd.appsGetterFunc(
		appsParser,
		&api.PaginatedRequester{
			RequestFactory: appRequestFactory,
			Client:         httpClient,
			PageParser:     pageParser,
		},
	)
	if err != nil {
		return err
	}

	spaceRequestFactory := apiClient.HandleFiltersAndParameters(
		apiClient.Authorize(apiClient.NewGetSpacesRequest),
	)

	spaces, err := thingdoer.Spaces(
		spacesParser,
		&api.PaginatedRequester{
			RequestFactory: spaceRequestFactory,
			Client:         httpClient,
			PageParser:     pageParser,
		},
	)
	if err != nil {
		return err
	}

	spaceMap := make(map[string]models.Space)
	for _, space := range spaces {
		spaceMap[space.Guid] = space
	}

	diegoSupport := diego_support.NewDiegoSupport(cliConnection)

	warnings := 0
	for _, app := range apps {
		a := &appPrinter{
			app:    app,
			spaces: spaceMap,
		}

		cmd.migrateAppsCommand.BeforeEach(a)

		var waitTime time.Duration
		if app.State == models.Started {
			waitTime = 1 * time.Minute
			timeout := os.Getenv("CF_STARTUP_TIMEOUT")
			if timeout != "" {
				t, err := strconv.Atoi(timeout)

				if err == nil {
					waitTime = time.Duration(float32(t)/5.0*60.0) * time.Second
				}
			}
		}

		_, err := diegoSupport.SetDiegoFlag(app.Guid, cmd.runtime == ui.Diego)
		if err != nil {
			warnings += 1
			fmt.Println("Error: ", err)
			fmt.Println("Continuing...")
			// WARNING: No authorization to migrate app APP_NAME in org ORG_NAME / space SPACE_NAME to RUNTIME as PERSON...
			continue
		}

		printDot := time.NewTicker(5 * time.Second)
		go func() {
			for range printDot.C {
				cmd.migrateAppsCommand.DuringEach(a)
			}
		}()
		time.Sleep(waitTime)
		printDot.Stop()

		cmd.migrateAppsCommand.CompletedEach(a)
	}

	cmd.migrateAppsCommand.AfterAll(len(apps), warnings)
	return nil
}

func newMigrateAppsCommand(cliConnection plugin.CliConnection, opts Opts, runtime ui.Runtime) (ui.MigrateAppsCommand, error) {
	username, err := cliConnection.Username()
	if err != nil {
		return ui.MigrateAppsCommand{}, err
	}

	return ui.MigrateAppsCommand{
		Username:     username,
		Organization: opts.Organization,
		Runtime:      runtime,
	}, nil
}
