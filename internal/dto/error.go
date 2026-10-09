package dto

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Error struct {
	Error ErrorDetail `json:"error"`
}
