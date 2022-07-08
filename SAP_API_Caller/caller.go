package sap_api_caller

import (
	"fmt"
	"io/ioutil"
	"net/http"
	sap_api_output_formatter "sap-api-integrations-chat-activity-reads-rmq-kube/SAP_API_Output_Formatter"
	"strings"
	"sync"

	"github.com/latonaio/golang-logging-library-for-sap/logger"
	"golang.org/x/xerrors"
)

type RMQOutputter interface {
	Send(sendQueue string, payload map[string]interface{}) error
}

type SAPAPICaller struct {
	baseURL      string
	apiKey       string
	outputQueues []string
	outputter    RMQOutputter
	log          *logger.Logger
}

func NewSAPAPICaller(baseUrl string, outputQueueTo []string, outputter RMQOutputter, l *logger.Logger) *SAPAPICaller {
	return &SAPAPICaller{
		baseURL:      baseUrl,
		apiKey:       GetApiKey(),
		outputQueues: outputQueueTo,
		outputter:    outputter,
		log:          l,
	}
}

func (c *SAPAPICaller) AsyncGetChatActivityCollection(iD, text string, accepter []string) {
	wg := &sync.WaitGroup{}
	wg.Add(len(accepter))
	for _, fn := range accepter {
		switch fn {
		case "ChatActivityCollection":
			func() {
				c.ChatActivityCollection(iD)
				wg.Done()
			}()
		case "ChatActivityTextCollection":
			func() {
				c.ChatActivityTextCollection(text)
				wg.Done()
			}()
		default:
			wg.Done()
		}
	}

	wg.Wait()
}

func (c *SAPAPICaller) ChatActivityCollection(iD string) {
	chatActivityCollectionData, err := c.callChatActivitySrvAPIRequirementChatActivityCollection("ChatActivityCollectionData", iD)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": chatActivityCollectionData, "function": "ChatActivityCollectionData"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(chatActivityCollectionData)

	chatActivityPartiesData, err := c.callChatActivityParties(chatActivityCollectionData[0].ToChatActivityParties)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": chatActivityPartiesData, "function": "ChatActivityPartiesData"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(chatActivityPartiesData)

}

func (c *SAPAPICaller) callChatActivitySrvAPIRequirementChatActivityCollection(api, iD string) ([]sap_api_output_formatter.ChatActivityCollection, error) {
	url := strings.Join([]string{c.baseURL, "c4codataapi", api}, "/")
	req, _ := http.NewRequest("GET", url, nil)

	c.setHeaderAPIKeyAccept(req)
	c.getQueryWithChatActivityCollection(req, iD)

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, xerrors.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	data, err := sap_api_output_formatter.ConvertToChatActivityCollection(byteArray, c.log)
	if err != nil {
		return nil, xerrors.Errorf("convert error: %w", err)
	}
	return data, nil
}

func (c *SAPAPICaller) callChatActivityParties(url string) ([]sap_api_output_formatter.ChatActivityParties, error) {
	req, _ := http.NewRequest("GET", url, nil)
	c.setHeaderAPIKeyAccept(req)

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, xerrors.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	data, err := sap_api_output_formatter.ConvertToChatActivityParties(byteArray, c.log)
	if err != nil {
		return nil, xerrors.Errorf("convert error: %w", err)
	}
	return data, nil
}

func (c *SAPAPICaller) ChatActivityTextCollection(text string) {
	chatActivityTextCollectionData, err := c.callChatActivitySrvAPIRequirementChatActivityTextCollection("ChatActivityTextCollectionData", text)
	if err != nil {
		c.log.Error(err)
		return
	}
	err = c.outputter.Send(c.outputQueues[0], map[string]interface{}{"message": chatActivityTextCollectionData, "function": "ChatActivityTextCollectionData"})
	if err != nil {
		c.log.Error(err)
		return
	}
	c.log.Info(chatActivityTextCollectionData)
}

func (c *SAPAPICaller) callChatActivitySrvAPIRequirementChatActivityTextCollection(api, text string) ([]sap_api_output_formatter.ChatActivityTextCollection, error) {
	url := strings.Join([]string{c.baseURL, "c4codataapi", api}, "/")
	req, _ := http.NewRequest("GET", url, nil)

	c.setHeaderAPIKeyAccept(req)
	c.getQueryWithChatActivityTextCollection(req, text)

	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, xerrors.Errorf("API request error: %w", err)
	}
	defer resp.Body.Close()

	byteArray, _ := ioutil.ReadAll(resp.Body)
	data, err := sap_api_output_formatter.ConvertToChatActivityTextCollection(byteArray, c.log)
	if err != nil {
		return nil, xerrors.Errorf("convert error: %w", err)
	}
	return data, nil
}

func (c *SAPAPICaller) setHeaderAPIKeyAccept(req *http.Request) {
	req.Header.Set("APIKey", c.apiKey)
	req.Header.Set("Accept", "application/json")
}

func (c *SAPAPICaller) getQueryWithChatActivityCollection(req *http.Request, iD string) {
	params := req.URL.Query()
	params.Add("$filter", fmt.Sprintf("ID eq '%s'", iD))
	req.URL.RawQuery = params.Encode()
}

func (c *SAPAPICaller) getQueryWithChatActivityTextCollection(req *http.Request, text string) {
	params := req.URL.Query()
	params.Add("$filter", fmt.Sprintf("substringof('%s', Text)", text))
	req.URL.RawQuery = params.Encode()
}
