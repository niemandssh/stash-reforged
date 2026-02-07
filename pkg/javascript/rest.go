package javascript

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dop251/goja"
)

// responseWriter is a minimal http.ResponseWriter that captures the response.
type responseWriter struct {
	header     http.Header
	r          bytes.Buffer
	statusCode int
}

func (w *responseWriter) Header() http.Header {
	return w.header
}

func (w *responseWriter) Write(b []byte) (int, error) {
	return w.r.Write(b)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

// REST provides REST API access to JavaScript plugins.
// It calls the REST handler directly (no network round-trip), similar to how
// GQL calls the GraphQL handler.
type REST struct {
	Context     context.Context
	Cookie      *http.Cookie
	RESTHandler http.Handler
}

func (r *REST) doRequest(vm *VM, method, path string, body interface{}) (goja.Value, error) {
	var bodyReader *bytes.Buffer
	if body != nil {
		bodyReader = &bytes.Buffer{}
		if err := json.NewEncoder(bodyReader).Encode(body); err != nil {
			return nil, fmt.Errorf("encoding request body: %w", err)
		}
	}

	var httpBody *bytes.Buffer
	if bodyReader != nil {
		httpBody = bodyReader
	}

	var req *http.Request
	var err error
	if httpBody != nil {
		req, err = http.NewRequestWithContext(r.Context, method, "/api/v1"+path, httpBody)
	} else {
		req, err = http.NewRequestWithContext(r.Context, method, "/api/v1"+path, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if r.Cookie != nil {
		req.AddCookie(r.Cookie)
	}

	w := &responseWriter{
		header: make(http.Header),
	}

	r.RESTHandler.ServeHTTP(w, req)

	output := w.r.String()

	if w.statusCode >= 400 {
		vm.Throw(fmt.Errorf("REST API error %d: %s", w.statusCode, output))
	}

	if output == "" {
		return goja.Null(), nil
	}

	var obj interface{}
	if err := json.Unmarshal([]byte(output), &obj); err != nil {
		vm.Throw(fmt.Errorf("could not unmarshal response %s: %w", output, err))
	}

	return vm.ToValue(obj), nil
}

func (r *REST) getFunc(vm *VM) func(path string) (goja.Value, error) {
	return func(path string) (goja.Value, error) {
		return r.doRequest(vm, "GET", path, nil)
	}
}

func (r *REST) postFunc(vm *VM) func(path string, body interface{}) (goja.Value, error) {
	return func(path string, body interface{}) (goja.Value, error) {
		return r.doRequest(vm, "POST", path, body)
	}
}

func (r *REST) putFunc(vm *VM) func(path string, body interface{}) (goja.Value, error) {
	return func(path string, body interface{}) (goja.Value, error) {
		return r.doRequest(vm, "PUT", path, body)
	}
}

func (r *REST) patchFunc(vm *VM) func(path string, body interface{}) (goja.Value, error) {
	return func(path string, body interface{}) (goja.Value, error) {
		return r.doRequest(vm, "PATCH", path, body)
	}
}

func (r *REST) deleteFunc(vm *VM) func(path string, body interface{}) (goja.Value, error) {
	return func(path string, body interface{}) (goja.Value, error) {
		return r.doRequest(vm, "DELETE", path, body)
	}
}

// AddToVM adds the REST API client to the JavaScript VM.
// Exposed as:
//
//	rest.Get(path)           - GET request
//	rest.Post(path, body)    - POST request
//	rest.Put(path, body)     - PUT request
//	rest.Patch(path, body)   - PATCH request
//	rest.Delete(path, body)  - DELETE request
func (r *REST) AddToVM(globalName string, vm *VM) error {
	restObj := vm.NewObject()

	if err := restObj.Set("Get", r.getFunc(vm)); err != nil {
		return fmt.Errorf("unable to set REST Get function: %w", err)
	}
	if err := restObj.Set("Post", r.postFunc(vm)); err != nil {
		return fmt.Errorf("unable to set REST Post function: %w", err)
	}
	if err := restObj.Set("Put", r.putFunc(vm)); err != nil {
		return fmt.Errorf("unable to set REST Put function: %w", err)
	}
	if err := restObj.Set("Patch", r.patchFunc(vm)); err != nil {
		return fmt.Errorf("unable to set REST Patch function: %w", err)
	}
	if err := restObj.Set("Delete", r.deleteFunc(vm)); err != nil {
		return fmt.Errorf("unable to set REST Delete function: %w", err)
	}

	if err := vm.Set(globalName, restObj); err != nil {
		return fmt.Errorf("unable to set rest: %w", err)
	}

	return nil
}
