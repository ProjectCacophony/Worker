package api

import (
	"reflect"
	"runtime"

	"net/http"

	"github.com/go-chi/chi"
	chiMiddleware "github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"gitlab.com/Cacophony/dhelpers"
	"gitlab.com/Cacophony/dhelpers/apihelper"
	"gitlab.com/Cacophony/dhelpers/cache"
	"gitlab.com/Cacophony/dhelpers/middleware"
)

// New creates a new restful Web Service for reporting information about the worker
func New() http.Handler {
	router := chi.NewRouter()

	// setup middleware
	chiMiddleware.DefaultLogger = chiMiddleware.RequestLogger(&chiMiddleware.DefaultLogFormatter{Logger: cache.GetLogger(), NoColor: false})
	router.Use(chiMiddleware.Logger)
	router.Use(middleware.Service("gateway"))
	router.Use(middleware.Recoverer)
	router.Use(chiMiddleware.DefaultCompress)
	router.Use(render.SetContentType(render.ContentTypeJSON))

	router.HandleFunc("/stats", getStats)

	return router
}

func getStats(w http.ResponseWriter, r *http.Request) {
	// gather data
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

	// return result
	err := render.Render(w, r, result)
	dhelpers.LogError(err)
}
