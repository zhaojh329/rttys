package pwauth

// Auth check the validity of the username/password pair.If the
// credentials are not valid, this function will return an error.
func Auth(username, password string) error {
	return auth(username, password)
}
