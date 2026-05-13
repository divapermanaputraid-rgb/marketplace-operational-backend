package models

// APIResponse is the standard response wrapper.
// All API endpoints should return this format for consistency.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError holds structured error information.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SuccessResponse creates a standard success response.
func SuccessResponse(data interface{}, message string) APIResponse {
	return APIResponse{
		Success: true,
		Data:    data,
		Message: message,
	}
}

// ErrorResponse creates a standard error response.
func ErrorResponse(code string, message string) APIResponse {
	return APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	}
}
