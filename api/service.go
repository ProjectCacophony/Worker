package api

import (
	"reflect"
	"runtime"

	"github.com/emicklei/go-restful"
	"gitlab.com/project-d-collab/dhelpers/apihelper"
	"gitlab.com/project-d-collab/dhelpers/cache"
)

// New creates a new restful Web Service for reporting information about the worker
func New() *restful.WebService {
	service := new(restful.WebService)
	service.
		Path("/stats").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	service.Route(service.GET("").To(getStats))

	return service
}

func getStats(_ *restful.Request, response *restful.Response) {
	var result apihelper.WorkerStatus
	for _, entry := range cache.GetCron().Entries() {
		result.Entries = append(result.Entries, apihelper.WorkerJobInformation{
			Function: runtime.FuncForPC(reflect.ValueOf(entry.Job).Pointer()).Name(),
			Next:     entry.Next,
			Prev:     entry.Prev,
		})
	}
	result.Service = apihelper.GenerateServiceInformation()
	result.Available = true
	response.WriteEntity(result) // nolint: errcheck, gas
}
