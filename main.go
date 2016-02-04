package main

type fakeUserProvider struct {
}

func (a fakeUserProvider) login(username string, password string) (result bool) {
	if username == "testuser" && password == "testpassword" {
		return true
	}
	return false
}

func main() {
	var provider = fakeUserProvider{}
	api := CreateRetwisApi(provider)
	api.RunServer()
}
