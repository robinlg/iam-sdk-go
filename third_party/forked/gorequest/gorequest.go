package gorequest

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/textproto"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
	"moul.io/http2curl"
)

type Request *http.Request
type Response *http.Response

// HTTP methods we support
const (
	POST    = "POST"
	GET     = "GET"
	HEAD    = "HEAD"
	PUT     = "PUT"
	DELETE  = "DELETE"
	PATCH   = "PATCH"
	OPTIONS = "OPTIONS"
)

// Types we support.
const (
	TypeJSON       = "json"
	TypeXML        = "xml"
	TypeUrlencoded = "urlencoded"
	TypeForm       = "form"
	TypeFormData   = "form-data"
	TypeHTML       = "html"
	TypeText       = "text"
	TypeMultipart  = "multipart"
)

type superAgentRetryable struct {
	RetryableStatus []int
	RetryerTime     time.Duration
	RetryerCount    int
	Attempt         int
	Enable          bool
}

// A SuperAgent is a object storing all request data for client.
type SuperAgent struct {
	Url                  string
	Method               string
	Header               http.Header
	TargetType           string
	ForceType            string
	Data                 map[string]interface{}
	SliceData            []interface{}
	FormData             url.Values
	QueryData            url.Values
	FileData             []File
	BounceToRawString    bool
	RawString            string
	Client               *http.Client
	Transport            *http.Transport
	Cookies              []*http.Cookie
	Errors               []error
	BasicAuth            struct{ Username, Password string }
	Debug                bool
	CurlCommand          bool
	logger               Logger
	Retryable            superAgentRetryable
	DoNotClearSuperAgent bool
	isClone              bool
	ctx                  context.Context
}

var DisableTransportSwap = false

// Used to create a new SuperAgent object.
func New() *SuperAgent {
	cookiejarOptions := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, _ := cookiejar.New(&cookiejarOptions)

	debug := os.Getenv("GOREQUEST_DEBUG") == "1"

	s := &SuperAgent{
		TargetType:        TypeJSON,
		Data:              make(map[string]interface{}),
		Header:            http.Header{},
		RawString:         "",
		SliceData:         []interface{}{},
		FormData:          url.Values{},
		QueryData:         url.Values{},
		FileData:          make([]File, 0),
		BounceToRawString: false,
		Client:            &http.Client{Jar: jar},
		Transport:         &http.Transport{},
		Cookies:           make([]*http.Cookie, 0),
		Errors:            nil,
		BasicAuth:         struct{ Username, Password string }{},
		Debug:             debug,
		CurlCommand:       false,
		logger:            log.New(os.Stderr, "[gorequest]", log.LstdFlags),
		isClone:           false,
	}
	// disable keep alives by default, see this issue https://github.com/parnurzeal/gorequest/issues/75
	s.Transport.DisableKeepAlives = true
	return s
}

// Clear SuperAgent data for another new request.
func (s *SuperAgent) ClearSuperAgent() {
	return
	if s.DoNotClearSuperAgent {
		return
	}
	s.Url = ""
	s.Method = ""
	s.Header = http.Header{}
	s.Data = make(map[string]interface{})
	s.SliceData = []interface{}{}
	s.FormData = url.Values{}
	s.QueryData = url.Values{}
	s.FileData = make([]File, 0)
	s.BounceToRawString = false
	s.RawString = ""
	s.ForceType = ""
	s.TargetType = TypeJSON
	s.Cookies = make([]*http.Cookie, 0)
	s.Errors = nil
}

// Just a wrapper to initialize SuperAgent instance by method string
func (s *SuperAgent) CustomMethod(method, targetUrl string) *SuperAgent {
	switch method {
	case POST:
		return s.Post(targetUrl)
	case GET:
		return s.Get(targetUrl)
	case HEAD:
		return s.Head(targetUrl)
	case PUT:
		return s.Put(targetUrl)
	case DELETE:
		return s.Delete(targetUrl)
	case PATCH:
		return s.Patch(targetUrl)
	case OPTIONS:
		return s.Options(targetUrl)
	default:
		s.ClearSuperAgent()
		s.Method = method
		s.Url = targetUrl
		s.Errors = nil
		return s
	}
}

func (s *SuperAgent) Get(targetUrl string) *SuperAgent {
	s.ClearSuperAgent()
	s.Method = GET
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *SuperAgent) Post(targetUrl string) *SuperAgent {
	s.ClearSuperAgent()
	s.Method = POST
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *SuperAgent) Head(targetUrl string) *SuperAgent {
	s.ClearSuperAgent()
	s.Method = HEAD
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *SuperAgent) Put(targetUrl string) *SuperAgent {
	s.ClearSuperAgent()
	s.Method = PUT
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *SuperAgent) Delete(targetUrl string) *SuperAgent {
	s.ClearSuperAgent()
	s.Method = DELETE
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *SuperAgent) Patch(targetUrl string) *SuperAgent {
	s.ClearSuperAgent()
	s.Method = PATCH
	s.Url = targetUrl
	s.Errors = nil
	return s
}

func (s *SuperAgent) Options(targetUrl string) *SuperAgent {
	s.ClearSuperAgent()
	s.Method = OPTIONS
	s.Url = targetUrl
	s.Errors = nil
	return s
}

// gorequest.New().
//
//	Post("/gamelist").
//	Retry(3, 5 * time.seconds, http.StatusBadRequest, http.StatusInternalServerError).
//	End()
func (s *SuperAgent) Retry(retryerCount int, retryerTime time.Duration, statusCode ...int) *SuperAgent {
	for _, code := range statusCode {
		statusText := http.StatusText(code)
		if len(statusText) == 0 {
			s.Errors = append(s.Errors, errors.New("StatusCode '"+strconv.Itoa(code)+"' doesn't exist in http package"))
		}
	}

	s.Retryable = struct {
		RetryableStatus []int
		RetryerTime     time.Duration
		RetryerCount    int
		Attempt         int
		Enable          bool
	}{
		statusCode,
		retryerTime,
		retryerCount,
		0,
		true,
	}
	return s
}

var Types = map[string]string{
	TypeJSON:       "application/json",
	TypeXML:        "application/xml",
	TypeForm:       "application/x-www-form-urlencoded",
	TypeFormData:   "application/x-www-form-urlencoded",
	TypeUrlencoded: "application/x-www-form-urlencoded",
	TypeHTML:       "text/html",
	TypeText:       "text/plain",
	TypeMultipart:  "multipart/form-data",
}

// Type is a convenience function to specify the data type to send.
// For example, to send data as `application/x-www-form-urlencoded` :
//
//	gorequest.New().
//	  Post("/recipe").
//	  Type("form").
//	  Send(`{ "name": "egg benedict", "category": "brunch" }`).
//	  End()
//
// This will POST the body "name=egg benedict&category=brunch" to url /recipe
//
// GoRequest supports
//
//	"text/html" uses "html"
//	"application/json" uses "json"
//	"application/xml" uses "xml"
//	"text/plain" uses "text"
//	"application/x-www-form-urlencoded" uses "urlencoded", "form" or "form-data"
func (s *SuperAgent) Type(typeStr string) *SuperAgent {
	if _, ok := Types[typeStr]; ok {
		s.ForceType = typeStr
	} else {
		s.Errors = append(s.Errors, errors.New("Type func: incorrect type \""+typeStr+"\""))
	}
	return s
}

// Set TLSClientConfig for underling Transport.
// One example is you can use it to disable security check (https):
//
//	gorequest.New().TLSClientConfig(&tls.Config{ InsecureSkipVerify: true}).
//	  Get("https://disable-security-check.com").
//	  End()
func (s *SuperAgent) TLSClientConfig(config *tls.Config) *SuperAgent {
	s.safeModifyTransport()
	s.Transport.TLSClientConfig = config
	return s
}

type File struct {
	Filename  string
	Fieldname string
	Data      []byte
}

func changeMapToURLValues(data map[string]interface{}) url.Values {
	var newUrlValues = url.Values{}
	for k, v := range data {
		switch val := v.(type) {
		case string:
			newUrlValues.Add(k, val)
		case bool:
			newUrlValues.Add(k, strconv.FormatBool(val))
		// if a number, change to string
		// json.Number used to protect against a wrong (for GoRequest) default conversion
		// which always converts number to float64.
		// This type is caused by using Decoder.UseNumber()
		case json.Number:
			newUrlValues.Add(k, string(val))
		case int:
			newUrlValues.Add(k, strconv.FormatInt(int64(val), 10))
		// TODO add all other int-Types (int8, int16, ...)
		case float64:
			newUrlValues.Add(k, strconv.FormatFloat(float64(val), 'f', -1, 64))
		case float32:
			newUrlValues.Add(k, strconv.FormatFloat(float64(val), 'f', -1, 64))
		// following slices are mostly needed for tests
		case []string:
			for _, element := range val {
				newUrlValues.Add(k, element)
			}
		case []int:
			for _, element := range val {
				newUrlValues.Add(k, strconv.FormatInt(int64(element), 10))
			}
		case []bool:
			for _, element := range val {
				newUrlValues.Add(k, strconv.FormatBool(element))
			}
		case []float64:
			for _, element := range val {
				newUrlValues.Add(k, strconv.FormatFloat(float64(element), 'f', -1, 64))
			}
		case []float32:
			for _, element := range val {
				newUrlValues.Add(k, strconv.FormatFloat(float64(element), 'f', -1, 64))
			}
		// these slices are used in practice like sending a struct
		case []interface{}:

			if len(val) <= 0 {
				continue
			}

			switch val[0].(type) {
			case string:
				for _, element := range val {
					newUrlValues.Add(k, element.(string))
				}
			case bool:
				for _, element := range val {
					newUrlValues.Add(k, strconv.FormatBool(element.(bool)))
				}
			case json.Number:
				for _, element := range val {
					newUrlValues.Add(k, string(element.(json.Number)))
				}
			}
		default:
			// TODO add ptr, arrays, ...
		}
	}
	return newUrlValues
}

// End is the most important function that you need to call when ending the chain. The request won't proceed without
// calling it.
// End function returns Response which matchs the structure of Response type in Golang's http package (but without Body
// data). The body data itself returns as a string in a 2nd return value.
// Lastly but worth noticing, error array (NOTE: not just single error value) is returned as a 3rd value and nil
// otherwise.
//
// For example:
//
//	resp, body, errs := gorequest.New().Get("http://www.google.com").End()
//	if errs != nil {
//	  fmt.Println(errs)
//	}
//	fmt.Println(resp, body)
//
// Moreover, End function also supports callback which you can put as a parameter.
// This extends the flexibility and makes GoRequest fun and clean! You can use GoRequest in whatever style you love!
//
// For example:
//
//	func printBody(resp gorequest.Response, body string, errs []error){
//	  fmt.Println(resp.Status)
//	}
//	gorequest.New().Get("http://www..google.com").End(printBody)
func (s *SuperAgent) End(callback ...func(response Response, body string, errs []error)) (Response, string, []error) {
	var bytesCallback []func(response Response, body []byte, errs []error)
	if len(callback) > 0 {
		bytesCallback = []func(response Response, body []byte, errs []error){
			func(response Response, body []byte, errs []error) {
				callback[0](response, string(body), errs)
			},
		}
	}

	resp, body, errs := s.EndBytes(bytesCallback...)
	bodyString := string(body)

	return resp, bodyString, errs
}

// EndBytes should be used when you want the body as bytes. The callbacks work the same way as with `End`, except that a
// byte array is used instead of a string.
func (s *SuperAgent) EndBytes(
	callback ...func(response Response, body []byte, errs []error),
) (Response, []byte, []error) {
	var (
		errs []error
		resp Response
		body []byte
	)

	for {
		resp, body, errs = s.getResponseBytes()
		if errs != nil {
			return nil, nil, errs
		}
		if s.isRetryableRequest(resp) {
			resp.Header.Set("Retry-Count", strconv.Itoa(s.Retryable.Attempt))
			break
		}
	}

	respCallback := *resp
	if len(callback) != 0 {
		callback[0](&respCallback, body, s.Errors)
	}
	return resp, body, nil
}

func (s *SuperAgent) isRetryableRequest(resp Response) bool {
	if s.Retryable.Enable && s.Retryable.Attempt < s.Retryable.RetryerCount &&
		contains(resp.StatusCode, s.Retryable.RetryableStatus) {
		time.Sleep(s.Retryable.RetryerTime)
		s.Retryable.Attempt++
		return false
	}
	return true
}

func contains(respStatus int, statuses []int) bool {
	for _, status := range statuses {
		if status == respStatus {
			return true
		}
	}
	return false
}

func (s *SuperAgent) getResponseBytes() (Response, []byte, []error) {
	var (
		req  *http.Request
		err  error
		resp Response
	)
	// check whether there is an error. if yes, return all errors
	if len(s.Errors) != 0 {
		return nil, nil, s.Errors
	}
	// check if there is forced type
	switch s.ForceType {
	case TypeJSON, TypeForm, TypeXML, TypeText, TypeMultipart:
		s.TargetType = s.ForceType
		// If forcetype is not set, check whether user set Content-Type header.
		// If yes, also bounce to the correct supported TargetType automatically.
	default:
		contentType := s.Header.Get("Content-Type")
		for k, v := range Types {
			if contentType == v {
				s.TargetType = k
			}
		}
	}

	// if slice and map get mixed, let's bounce to rawstring
	if len(s.Data) != 0 && len(s.SliceData) != 0 {
		s.BounceToRawString = true
	}

	// Make Request
	req, err = s.MakeRequest()
	if err != nil {
		s.Errors = append(s.Errors, err)
		return nil, nil, s.Errors
	}

	// Set Transport
	if !DisableTransportSwap {
		s.Client.Transport = s.Transport
	}

	// Log details of this request
	if s.Debug {
		dump, err := httputil.DumpRequest(req, true)
		s.logger.SetPrefix("[http] ")
		if err != nil {
			s.logger.Println("Error:", err)
		} else {
			s.logger.Printf("HTTP Request: %s", string(dump))
		}
	}

	// Display CURL command line
	if s.CurlCommand {
		curl, err := http2curl.GetCurlCommand(req)
		s.logger.SetPrefix("[curl] ")
		if err != nil {
			s.logger.Println("Error:", err)
		} else {
			s.logger.Printf("CURL command line: %s", curl)
		}
	}

	// Send request
	resp, err = s.Client.Do(req)
	if err != nil {
		s.Errors = append(s.Errors, err)
		return nil, nil, s.Errors
	}
	defer resp.Body.Close()

	// Log details of this response
	if s.Debug {
		dump, err := httputil.DumpResponse(resp, true)
		if nil != err {
			s.logger.Println("Error:", err)
		} else {
			s.logger.Printf("HTTP Response: %s", string(dump))
		}
	}

	body, err := ioutil.ReadAll(resp.Body)
	// Reset resp.Body so it can be use again
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	if err != nil {
		return nil, nil, []error{err}
	}
	return resp, body, nil
}

func (s *SuperAgent) MakeRequest() (*http.Request, error) {
	var (
		req           *http.Request
		contentType   string // This is only set when the request body content is non-empty.
		contentReader io.Reader
		err           error
	)

	if s.Method == "" {
		return nil, errors.New("No method specified")
	}

	// !!! Important Note !!!
	//
	// Throughout this region, contentReader and contentType are only set when
	// the contents will be non-empty.
	// This is done avoid ever sending a non-nil request body with nil contents
	// to http.NewRequest, because it contains logic which dependends on
	// whether or not the body is "nil".
	//
	// See PR #136 for more information:
	//
	//     https://github.com/parnurzeal/gorequest/pull/136
	//
	switch s.TargetType {
	case TypeJSON:
		// If-case to give support to json array. we check if
		// 1) Map only: send it as json map from s.Data
		// 2) Array or Mix of map & array or others: send it as rawstring from s.RawString
		var contentJson []byte
		if s.BounceToRawString {
			contentJson = []byte(s.RawString)
		} else if len(s.Data) != 0 {
			contentJson, _ = json.Marshal(s.Data)
		} else if len(s.SliceData) != 0 {
			contentJson, _ = json.Marshal(s.SliceData)
		}
		if contentJson != nil {
			contentReader = bytes.NewReader(contentJson)
			contentType = "application/json"
		}
	case TypeForm, TypeFormData, TypeUrlencoded:
		var contentForm []byte
		if s.BounceToRawString || len(s.SliceData) != 0 {
			contentForm = []byte(s.RawString)
		} else {
			formData := changeMapToURLValues(s.Data)
			contentForm = []byte(formData.Encode())
		}
		if len(contentForm) != 0 {
			contentReader = bytes.NewReader(contentForm)
			contentType = "application/x-www-form-urlencoded"
		}
	case TypeText:
		if len(s.RawString) != 0 {
			contentReader = strings.NewReader(s.RawString)
			contentType = "text/plain"
		}
	case TypeXML:
		if len(s.RawString) != 0 {
			contentReader = strings.NewReader(s.RawString)
			contentType = "application/xml"
		}
	case TypeMultipart:
		var (
			buf = &bytes.Buffer{}
			mw  = multipart.NewWriter(buf)
		)

		if s.BounceToRawString {
			fieldName := s.Header.Get("data_fieldname")
			if fieldName == "" {
				fieldName = "data"
			}
			fw, _ := mw.CreateFormField(fieldName)
			fw.Write([]byte(s.RawString))
			contentReader = buf
		}

		if len(s.Data) != 0 {
			formData := changeMapToURLValues(s.Data)
			for key, values := range formData {
				for _, value := range values {
					fw, _ := mw.CreateFormField(key)
					fw.Write([]byte(value))
				}
			}
			contentReader = buf
		}

		if len(s.SliceData) != 0 {
			fieldName := s.Header.Get("json_fieldname")
			if fieldName == "" {
				fieldName = "data"
			}
			// copied from CreateFormField() in mime/multipart/writer.go
			h := make(textproto.MIMEHeader)
			fieldName = strings.Replace(strings.Replace(fieldName, "\\", "\\\\", -1), `"`, "\\\"", -1)
			h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"`, fieldName))
			h.Set("Content-Type", "application/json")
			fw, _ := mw.CreatePart(h)
			contentJson, err := json.Marshal(s.SliceData)
			if err != nil {
				return nil, err
			}
			fw.Write(contentJson)
			contentReader = buf
		}

		// add the files
		if len(s.FileData) != 0 {
			for _, file := range s.FileData {
				fw, _ := mw.CreateFormFile(file.Fieldname, file.Filename)
				fw.Write(file.Data)
			}
			contentReader = buf
		}

		// close before call to FormDataContentType ! otherwise its not valid multipart
		mw.Close()

		if contentReader != nil {
			contentType = mw.FormDataContentType()
		}
	default:
		// let's return an error instead of an nil pointer exception here
		return nil, errors.New("TargetType '" + s.TargetType + "' could not be determined")
	}

	if req, err = http.NewRequest(s.Method, s.Url, contentReader); err != nil {
		return nil, err
	}

	if s.ctx != nil {
		req.WithContext(s.ctx)
	}

	for k, vals := range s.Header {
		for _, v := range vals {
			req.Header.Add(k, v)
		}

		// Setting the Host header is a special case, see this issue: https://github.com/golang/go/issues/7682
		if strings.EqualFold(k, "Host") {
			req.Host = vals[0]
		}
	}

	// https://github.com/parnurzeal/gorequest/issues/164
	// Don't infer the content type header if an overrride is already provided.
	if len(contentType) != 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Add all querystring from Query func
	q := req.URL.Query()
	for k, v := range s.QueryData {
		for _, vv := range v {
			q.Add(k, vv)
		}
	}
	req.URL.RawQuery = q.Encode()

	// Add basic auth
	if s.BasicAuth != struct{ Username, Password string }{} {
		req.SetBasicAuth(s.BasicAuth.Username, s.BasicAuth.Password)
	}

	// Add cookies
	for _, cookie := range s.Cookies {
		req.AddCookie(cookie)
	}

	return req, nil
}
