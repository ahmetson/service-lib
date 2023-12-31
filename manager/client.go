package manager

import (
	"fmt"
	"github.com/ahmetson/client-lib"
	clientConfig "github.com/ahmetson/client-lib/config"
	serviceConfig "github.com/ahmetson/config-lib/service"
	"github.com/ahmetson/datatype-lib/data_type/key_value"
	"github.com/ahmetson/datatype-lib/message"
	handlerConfig "github.com/ahmetson/handler-lib/config"
)

//
// Interact with the service manager
//

type Client struct {
	*client.Socket
}

// NewClient returns a manager client based on the configuration
func NewClient(c *clientConfig.Client) (*Client, error) {
	socket, err := client.New(c)
	if err != nil {
		return nil, fmt.Errorf("client.New: %w", err)
	}

	return &Client{socket}, nil
}

// Heartbeat sends a command to the parent to make sure that it's live
func (c *Client) Heartbeat() error {
	req := &message.Request{
		Command:    Heartbeat,
		Parameters: key_value.New(),
	}

	reply, err := c.Request(req)
	if err != nil {
		return fmt.Errorf("c.Request(%s): %w", Heartbeat, err)
	}

	if !reply.IsOK() {
		return fmt.Errorf("reply error message: %s", reply.ErrorMessage())
	}

	return nil
}

func (c *Client) Close() error {
	req := &message.Request{
		Command:    Close,
		Parameters: key_value.New(),
	}

	err := c.Submit(req)
	if err != nil {
		return fmt.Errorf("c.Request(%s): %w", Heartbeat, err)
	}

	return nil
}

func (c *Client) ProxyChainsByLastProxy(proxyId string) ([]*serviceConfig.ProxyChain, error) {
	req := &message.Request{
		Command:    ProxyChainsByLastId,
		Parameters: key_value.New().Set("id", proxyId),
	}
	reply, err := c.Request(req)
	if err != nil {
		return nil, fmt.Errorf("c.Request: %w", err)
	}
	if !reply.IsOK() {
		return nil, fmt.Errorf("reply error message: %s", reply.ErrorMessage())
	}

	kvList, err := reply.ReplyParameters().NestedListValue("proxy_chains")
	if err != nil {
		return nil, fmt.Errorf("reply.ReplyParameters().NestedKeyValueList('proxy_chains'): %w", err)
	}

	proxyChains := make([]*serviceConfig.ProxyChain, len(kvList))
	for i, kv := range kvList {
		var proxyChain serviceConfig.ProxyChain
		err = kv.Interface(&proxyChain)
		if err != nil {
			return nil, fmt.Errorf("kv.Interface(proxyChains[%d]): %w", i, err)
		}
		proxyChains[i] = &proxyChain
	}

	return proxyChains, nil
}

// The Units method returns the destination units by a rule.
func (c *Client) Units(rule *serviceConfig.Rule) ([]*serviceConfig.Unit, error) {
	req := &message.Request{
		Command:    Units,
		Parameters: key_value.New().Set("rule", rule),
	}
	reply, err := c.Request(req)
	if err != nil {
		return nil, fmt.Errorf("c.Request: %w", err)
	}
	if !reply.IsOK() {
		return nil, fmt.Errorf("reply error message: %s", reply.ErrorMessage())
	}

	rawUnits, err := reply.ReplyParameters().NestedListValue("units")
	if err != nil {
		return nil, fmt.Errorf("reply.ReplyParameters().NestedKeyValueList('proxy_chains'): %w", err)
	}

	units := make([]*serviceConfig.Unit, len(rawUnits))
	for i, rawUnit := range rawUnits {
		var unit serviceConfig.Unit
		err = rawUnit.Interface(&unit)
		if err != nil {
			return nil, fmt.Errorf("rawUnits[%d].Interface: %w", i, err)
		}

		units[i] = &unit
	}

	return units, nil
}

// The HandlersByCategory returns the list of handlers filtered by the category
func (c *Client) HandlersByCategory(category string) ([]*handlerConfig.Handler, error) {
	if len(category) == 0 {
		return nil, fmt.Errorf("the 'category' parameter can not be empty")
	}

	req := &message.Request{
		Command:    HandlersByCategory,
		Parameters: key_value.New().Set("category", category),
	}
	reply, err := c.Request(req)
	if err != nil {
		return nil, fmt.Errorf("c.Request: %w", err)
	}
	if !reply.IsOK() {
		return nil, fmt.Errorf("reply error message: %s", reply.ErrorMessage())
	}

	rawConfigs, err := reply.ReplyParameters().NestedListValue("handler_configs")
	if err != nil {
		return nil, fmt.Errorf("reply.ReplyParameters().NestedKeyValueList('handler_configs'): %w", err)
	}

	configs := make([]*handlerConfig.Handler, len(rawConfigs))
	for i, rawConfig := range rawConfigs {
		var c handlerConfig.Handler
		err = rawConfig.Interface(&c)
		if err != nil {
			return nil, fmt.Errorf("rawConfigs[%d].Interface: %w", i, err)
		}

		configs[i] = &c
	}

	return configs, nil
}

// The HandlersByRule method returns the handler configs that matches to the destination rule
func (c *Client) HandlersByRule(rule *serviceConfig.Rule) ([]*handlerConfig.Handler, error) {
	req := &message.Request{
		Command:    HandlersByRule,
		Parameters: key_value.New().Set("rule", rule),
	}
	reply, err := c.Request(req)
	if err != nil {
		return nil, fmt.Errorf("c.Request: %w", err)
	}
	if !reply.IsOK() {
		return nil, fmt.Errorf("reply error message: %s", reply.ErrorMessage())
	}

	rawConfigs, err := reply.ReplyParameters().NestedListValue("handler_configs")
	if err != nil {
		return nil, fmt.Errorf("reply.ReplyParameters().NestedKeyValueList('handler_configs'): %w", err)
	}

	configs := make([]*handlerConfig.Handler, len(rawConfigs))
	for i, rawConfig := range rawConfigs {
		var c handlerConfig.Handler
		err = rawConfig.Interface(&c)
		if err != nil {
			return nil, fmt.Errorf("rawUnits[%d].Interface: %w", i, err)
		}

		configs[i] = &c
	}

	return configs, nil
}

// The ProxyConfigSet method tells to the parent that proxy configuration set
func (c *Client) ProxyConfigSet(rule *serviceConfig.Rule, serviceSource *serviceConfig.SourceService) error {
	req := &message.Request{
		Command:    ProxyConfigSet,
		Parameters: key_value.New().Set("rule", rule).Set("source_service", serviceSource),
	}
	reply, err := c.Request(req)
	if err != nil {
		return fmt.Errorf("c.Request: %w", err)
	}
	if !reply.IsOK() {
		return fmt.Errorf("reply error message: %s", reply.ErrorMessage())
	}

	return nil
}
