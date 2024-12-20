package cas

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
)

// NewServiceTicketValidator create a new *ServiceTicketValidator
func NewServiceTicketValidator(client *http.Client, casURL *url.URL) *ServiceTicketValidator {
	return &ServiceTicketValidator{
		client: client,
		casURL: casURL,
	}
}

// ServiceTicketValidator is responsible for the validation of a service ticket
type ServiceTicketValidator struct {
	client *http.Client
	casURL *url.URL
}

// ValidateTicket validates the service ticket for the given server. The method will try to use the service validate
// endpoint of the cas >= 2 protocol, if the service validate endpoint not available, the function will use the cas 1
// validate endpoint.
func (validator *ServiceTicketValidator) ValidateTicket(serviceURL *url.URL, ticket string) (*AuthenticationResponse, error) {
	slog.Info("cas: validating ticket", slog.Any("ticket", ticket), slog.Any("service", serviceURL))

	u, err := validator.ServiceValidateUrl(serviceURL, ticket)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("User-Agent", "Golang CAS client gopkg.in/cas")

	slog.Info("cas: attempting ticket validation", slog.Any("url", r.URL))

	resp, err := validator.client.Do(r)
	if err != nil {
		return nil, err
	}

	slog.Info("cas: request returned", slog.Any("method", r.Method), slog.Any("url", r.URL), slog.Any("status", resp.Status))

	if resp.StatusCode == http.StatusNotFound {
		return validator.validateTicketCas1(serviceURL, ticket)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cas: validate ticket: %v", string(body))
	}

	slog.Info("cas: received authentication response", slog.Any("response", string(body)))

	success, err := ParseServiceResponse(body)
	if err != nil {
		return nil, err
	}

	slog.Info("cas: parsed service response", slog.Any("response", success))

	return success, nil
}

// ServiceValidateUrl creates the service validation url for the cas >= 2 protocol.
// TODO the function is only exposed, because of the clients ServiceValidateUrl function
func (validator *ServiceTicketValidator) ServiceValidateUrl(serviceURL *url.URL, ticket string) (string, error) {
	u, err := validator.casURL.Parse(path.Join(validator.casURL.Path, "serviceValidate"))
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Add("service", sanitisedURLString(serviceURL))
	q.Add("ticket", ticket)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func (validator *ServiceTicketValidator) validateTicketCas1(serviceURL *url.URL, ticket string) (*AuthenticationResponse, error) {
	u, err := validator.ValidateUrl(serviceURL, ticket)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("User-Agent", "Golang CAS client gopkg.in/cas")

	slog.Info("cas: attempting ticket validation", slog.Any("url", r.URL))

	resp, err := validator.client.Do(r)
	if err != nil {
		return nil, err
	}

	slog.Info("cas: request returned", slog.Any("method", r.Method), slog.Any("url", r.URL), slog.Any("status", resp.Status))

	data, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		return nil, err
	}

	body := string(data)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cas: validate ticket: %v", body)
	}

	slog.Info("cas: received authentication response", slog.Any("response", body))

	if body == "no\n\n" {
		return nil, nil // not logged in
	}

	success := &AuthenticationResponse{
		User: body[4 : len(body)-1],
	}

	slog.Info("cas: parsed service response", slog.Any("response", success))

	return success, nil
}

// ValidateUrl creates the validation url for the cas >= 1 protocol.
// TODO the function is only exposed, because of the clients ValidateUrl function
func (validator *ServiceTicketValidator) ValidateUrl(serviceURL *url.URL, ticket string) (string, error) {
	u, err := validator.casURL.Parse(path.Join(validator.casURL.Path, "validate"))
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Add("service", sanitisedURLString(serviceURL))
	q.Add("ticket", ticket)
	u.RawQuery = q.Encode()

	return u.String(), nil
}
