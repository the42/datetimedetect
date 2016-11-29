package main

// Copyright (c) Johann Höchtl 2016
//
// See LICENSE for License

// RESTful service to check for the existence of datetime information in a data stream

import (
	"log"
	"net/http"
	"os"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/swagger"

	"github.com/the42/datetimecheck"
)

// DateTimeCheckResponse
type DateTimeCheckResponse struct {
	Response struct {
		Occurence *datetimecheck.DateTimeCheckResponse
	}
}

type dtcheckservice struct {
	c *datetimecheck.DateTimeChecker
}

func (d *dtcheckservice) checkdatetime(request *restful.Request, response *restful.Response) {
	var mimet *string
	mt := request.HeaderParameter("Content-type")
	if len(mt) != 0 {
		mimet = &mt
	}

	r, err := d.c.ContainsDateTimeStream(request.Request.Body, mimet)
	if err != nil {
		log.Fatal(err)
	}

	result := DateTimeCheckResponse{}
	result.Response.Occurence = r
	response.WriteAsJson(result)
}

func main() {
	ws := new(restful.WebService).
		Produces(restful.MIME_JSON)

	//BEGIN: CORS support
	if enable_cors := os.Getenv("ENABLE_CORS"); enable_cors != "" {
		cors := restful.CrossOriginResourceSharing{
			ExposeHeaders:  []string{"X-My-Header"},
			AllowedHeaders: []string{"Content-Type", "Accept"},
			AllowedMethods: []string{"POST"},
			CookiesAllowed: false,
			Container:      restful.DefaultContainer}

		restful.DefaultContainer.Filter(cors.Filter)
		// Add container filter to respond to OPTIONS
		restful.DefaultContainer.Filter(restful.DefaultContainer.OPTIONSFilter)
	}
	//END: CORS support

	c := &dtcheckservice{}
	if d, err := datetimecheck.NewDateTimeChecker(datetimecheck.Checkupto); err == nil {
		c.c = d
	} else {
		log.Fatalf("Cannot create NewDateTimeChecker: %s\n", err.Error())
		return
	}

	ws.Route(ws.PUT("/checkdatetime").
		To(c.checkdatetime).
		Doc("check for the availability of datetime information in a data stream. Uses the HTTP-Header Content-type for improved checks. If Content-type is unset, a content type autodetect will be performed").
		Produces(restful.MIME_JSON).
		Returns(http.StatusOK, "success", DateTimeCheckResponse{}))
	restful.Add(ws)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	hostname := os.Getenv("HOSTNAME")

	config := swagger.Config{
		WebServices:     restful.DefaultContainer.RegisteredWebServices(),
		ApiPath:         "/apidocs/apidocs.json",
		SwaggerPath:     "/swagger/",
		SwaggerFilePath: "./swagger-ui/dist"}
	swagger.RegisterSwaggerService(config, restful.DefaultContainer)

	log.Fatal(http.ListenAndServe(hostname+":"+port, nil))
}
