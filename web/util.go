package web

import (
	"net/http"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/go-chi/chi"
)

// hydrateTemplate fills a templateData struct with generic stuff like Params, User and LoggedIn
func (rt *Web) hydrateTemplate(r *http.Request) templateData {
	user := common.UserFromContext(r)
	problem := common.ProblemFromContext(r)

	return templateData{
		Params:   globParams(r),
		User:     &user,
		LoggedIn: user.ID != 0,
		Problem:  &problem,
		Version:  common.Version,

		// HACK: Move this somewhere else
		ProblemEditor: common.IsRProblemEditor(r),
		TaskEditor:    common.IsRTaskEditor(r),
	}
}

func (rt *Web) isProblemAuthor(r *http.Request) bool {
	return true
}

func globParams(r *http.Request) map[string]string {
	ctx := chi.RouteContext(r.Context())
	params := make(map[string]string)
	for i := 0; i < len(ctx.URLParams.Keys); i++ {
		params[ctx.URLParams.Keys[i]] = ctx.URLParams.Values[i]
	}
	return params
}

type testDataType struct {
	In  string
	Out string
}

func (rt *Web) getTestData(test common.Test) testDataType {
	in, out, err := rt.dm.GetTest(test.ProblemID, test.VisibleID)
	if err != nil {
		in = []byte("err")
		out = []byte("err")
	}
	if len(in) > 1000000 { // 1Mb
		in = []byte("too long to show")
	}
	if len(out) > 1000000 { // 1Mb
		out = []byte("too long to show")
	}
	return testDataType{In: string(in), Out: string(out)}
}
