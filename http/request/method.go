package request

var (
	MethodPost   = "POST"
	MethodGet    = "GET"
	MethodPut    = "PUT"
	MethodDelete = "DELETE"
	MethodHead   = "HEAD"
	MethodOption = "OPTION"
)

var methodMap = map[string]struct{}{
	MethodPost:   {},
	MethodGet:    {},
	MethodPut:    {},
	MethodDelete: {},
	MethodHead:   {},
	MethodOption: {},
}
