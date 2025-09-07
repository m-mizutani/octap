package domain

import "github.com/m-mizutani/goerr/v2"

var (
	ErrAuthentication = goerr.New("authentication failed")
	ErrAPIRequest     = goerr.New("API request failed")
	ErrConfiguration  = goerr.New("configuration error")
	ErrRepository     = goerr.New("repository error")
)
