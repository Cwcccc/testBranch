package httpclient_go

import (
	"crypto/tls"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/huaweicloud/terraform-provider-huaweicloud/huaweicloud/config"
	"github.com/huaweicloud/terraform-provider-huaweicloud/vendor/github.com/chnsz/golangsdk"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	MethodGet    = "GET"
	MethodPost   = "POST"
	MethodPut    = "PUT"
	MethodDelete = "DELETE"
)

type HttpClientGo struct {
	Client    *golangsdk.ServiceClient
	Method    string
	Url       string
	Body      golangsdk.RequestOpts
	Header    map[string]string
	request   *http.Request
	Error     error
	Transport *http.Transport
}

func NewHttpClientGo(c *config.Config, product, region string) (*HttpClientGo, error) {
	client, err := c.NewServiceClient(product, region)
	if err != nil {
		return nil, err
	}
	return &HttpClientGo{
		Client: client,
		Body: golangsdk.RequestOpts{
			KeepResponseBody: true,
		},
	}, nil
}

func (client *HttpClientGo) WithMethod(method string) *HttpClientGo {
	client.Method = method
	return client
}

func (client *HttpClientGo) WithUrl(url string) *HttpClientGo {
	client.Url = client.Client.Endpoint + url
	return client
}

func (client *HttpClientGo) WithUrlWithoutEndpoint(cfg *config.Config, srv, region, path string) *HttpClientGo {
	endpoint := config.GetServiceEndpoint(cfg, srv, region)

	client.Url = endpoint + path
	return client
}

func (client *HttpClientGo) WithBody(body interface{}) *HttpClientGo {
	client.Body.JSONBody = body
	return client
}

func (client *HttpClientGo) WithHeader(header map[string]string) *HttpClientGo {
	if len(header) == 0 {
		return client
	}
	client.Header = header
	return client
}

func (client *HttpClientGo) Do() (*http.Response, error) {
	return client.Client.Request(client.Method, client.Url, &client.Body)
}

func (client *HttpClientGo) WithTransport() *HttpClientGo {
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		},
	}
	return client
}

func (client HttpClientGo) CheckDeletedDiag(d *schema.ResourceData, err error, response *http.Response, msg string) ([]byte, diag.Diagnostics) {
	if err != nil {
		return nil, diag.Errorf("%s: %s", msg, err)
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, diag.Errorf("error convert data %s: %s", string(body), err)
	}

	if response.StatusCode == 200 {
		return body, nil
	}

	if strings.Contains(string(body), "does not exist") {
		resourceID := d.Id()
		d.SetId("")
		return nil, diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Resource not found",
				Detail:   fmt.Sprintf("the resource %s is goneand will be removed in Terraform state.", resourceID),
			},
		}
	}
	return nil, diag.Errorf("%s: %s", msg, err)
}
