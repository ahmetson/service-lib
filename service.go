// Package service is the primary service.
// This package is calling out the orchestra. Then within that orchestra sets up
// - handler manager
// - proxies
// - extensions
// - config manager
// - dep manager
package service

import (
	"fmt"
	clientConfig "github.com/ahmetson/client-lib/config"
	serviceConfig "github.com/ahmetson/config-lib/service"
	"github.com/ahmetson/datatype-lib/data_type/key_value"
	context "github.com/ahmetson/dev-lib"
	"github.com/ahmetson/handler-lib/base"
	"github.com/ahmetson/handler-lib/manager_client"
	"github.com/ahmetson/log-lib"
	"github.com/ahmetson/os-lib/arg"
	"github.com/ahmetson/service-lib/flag"
	"github.com/ahmetson/service-lib/manager"
	"slices"
	"sync"
)

// Service keeps all necessary parameters of the service.
type Service struct {
	config             *serviceConfig.Service
	ctx                context.Interface // context handles the configuration and dependencies
	Handlers           key_value.KeyValue
	RequiredExtensions key_value.KeyValue
	Logger             *log.Logger
	Type               serviceConfig.Type
	id                 string
	url                string
	blocker            *sync.WaitGroup
	manager            *manager.Manager // manage this service from other parts
}

// New service.
// Requires url and id.
// The url and id could be passed as flag.IdFlag, flag.UrlFlag.
// Or url and id could be passed as environment variable flag.IdEnv, flag.UrlEnv.
//
// It will also create the context internally and start it.
func New() (*Service, error) {
	id := ""
	url := ""

	// let's validate the parameters of the service
	if arg.FlagExist(flag.IdFlag) {
		id = arg.FlagValue(flag.IdFlag)
	}
	if arg.FlagExist(flag.UrlFlag) {
		url = arg.FlagValue(flag.UrlFlag)
	}

	// Start the context
	ctx, err := context.New()
	if err != nil {
		return nil, fmt.Errorf("context.New: %w", err)
	}
	err = ctx.StartConfig()
	if err != nil {
		return nil, fmt.Errorf("ctx('%s').StartConfig: %w", ctx.Type(), err)
	}

	independent := &Service{
		ctx:      ctx,
		Handlers: key_value.New(),
		url:      url,
		id:       id,
		Type:     serviceConfig.IndependentType,
		blocker:  nil,
	}

	logger, err := log.New(id, true)
	if err != nil {
		err = fmt.Errorf("log.New(%s): %w", id, err)

		if closeErr := ctx.Close(); closeErr != nil {
			return nil, fmt.Errorf("%v: ctx.Close: %w", err, closeErr)
		}

		return nil, err
	}
	independent.Logger = logger

	if len(id) == 0 {
		configClient := ctx.Config()
		id, err = configClient.String(flag.IdEnv)
		if err != nil {
			err = fmt.Errorf("configClient.String('%s'): %w", flag.IdEnv, err)
			if closeErr := ctx.Close(); closeErr != nil {
				return nil, fmt.Errorf("%v: ctx.Close: %w", err, closeErr)
			}
			return nil, err
		}
	}
	if len(url) == 0 {
		configClient := ctx.Config()
		url, err = configClient.String(flag.UrlEnv)
		if err != nil {
			err = fmt.Errorf("configClient.String('%s'): %w", flag.UrlEnv, err)
			if closeErr := ctx.Close(); closeErr != nil {
				return nil, fmt.Errorf("%v: ctx.Close: %w", err, closeErr)
			}
			return nil, err
		}
	}

	if len(id) == 0 {
		err = fmt.Errorf("service can not identify itself. Either use %s flag or %s environment variable", flag.IdFlag, flag.IdEnv)
		if closeErr := ctx.Close(); closeErr != nil {
			return nil, fmt.Errorf("%v: ctx.Close: %w", err, closeErr)
		}
		return nil, err
	}
	if len(url) == 0 {
		err = fmt.Errorf("service can not identify it's class. Either use %s flag or %s environment variable", flag.UrlFlag, flag.UrlEnv)
		if closeErr := ctx.Close(); closeErr != nil {
			return nil, fmt.Errorf("%v: ctx.Close: %w", err, closeErr)
		}
		return nil, err
	}

	return independent, nil
}

// SetHandler of category
//
// Todo change to keep the handlers by their id.
func (independent *Service) SetHandler(category string, controller base.Interface) {
	independent.Handlers.Set(category, controller)
}

// SetTypeByService overwrites the type from the extended service.
// Proxy and Extension calls this function.
func (independent *Service) SetTypeByService(newType serviceConfig.Type) {
	independent.Type = newType
}

// Url returns the url of the service source code
func (independent *Service) Url() string {
	return independent.url
}

// Id returns the unique id of the service
func (independent *Service) Id() string {
	return independent.id
}

// SetProxyChain adds a proxy chain to the list of proxy chains to set.
//
// The proxies are managed by the proxy handler in the context.
// This method creates a serviceConfig.ProxyChain.
// Then send it to the proxy handler.
func (independent *Service) SetProxyChain(params ...interface{}) error {
	if len(params) < 2 || len(params) > 3 {
		return fmt.Errorf("argument amount is invalid, either two or three arguments must be set")
	}
	if independent.ctx == nil || independent.ctx.ProxyClient() == nil {
		return fmt.Errorf("context or proxy client are not set")
	}

	var sources []string
	if len(params) == 3 {
		src, ok := params[0].(string)
		if ok {
			sources = []string{src}
		} else {
			sourceUrls, ok := params[0].([]string)
			if !ok {
				return fmt.Errorf("first argument must be string or []string")
			} else {
				sources = sourceUrls
			}
		}
	}

	i := len(params) - 2
	var proxies []*serviceConfig.Proxy
	proxy, ok := params[i].(*serviceConfig.Proxy)
	if ok {
		proxies = []*serviceConfig.Proxy{proxy}
	} else {
		requiredProxies, ok := params[i].([]*serviceConfig.Proxy)
		if !ok {
			return fmt.Errorf("the second argument must be service.Proxy or []service.Proxy")
		}
		if len(requiredProxies) == 0 {
			return fmt.Errorf("proxy argument []service.Proxy has no element")
		}
		proxies = requiredProxies
	}

	i++
	var rule *serviceConfig.Rule
	requiredRule, ok := params[i].(*serviceConfig.Rule)
	if !ok {
		return fmt.Errorf("the third argument must be service.Rule")
	} else {
		rule = requiredRule
	}
	if len(rule.Urls) == 0 {
		rule.Urls = []string{independent.url}
	}

	proxyChain := &serviceConfig.ProxyChain{
		Sources:     sources,
		Proxies:     proxies,
		Destination: rule,
	}
	if !proxyChain.IsValid() {
		return fmt.Errorf("proxy chain not valid")
	}

	proxyClient := independent.ctx.ProxyClient()
	if err := proxyClient.Set(proxyChain); err != nil {
		return fmt.Errorf("c.Set: %w", err)
	}

	return nil
}

// RequireExtension lints the id to the extension url
func (independent *Service) RequireExtension(id string, url string) {
	if independent.RequiredExtensions.Exist(id) {
		independent.RequiredExtensions.Set(id, url)
	}
}

func (independent *Service) requiredControllerExtensions() []string {
	var extensions []string
	for _, controllerInterface := range independent.Handlers {
		c := controllerInterface.(base.Interface)
		extensions = append(extensions, c.DepIds()...)
	}

	return extensions
}

// generateConfig sends a signal to the context to generate a new configuration for this service.
// The method requests multiple commands. One command to generate a service configuration.
// Then a request to generate a handler configurations.
//
// The generated configuration returned back.
func (independent *Service) generateConfig() (*serviceConfig.Service, error) {
	configClient := independent.ctx.Config()

	generatedConfig, err := configClient.GenerateService(independent.id, independent.url, independent.Type)
	if err != nil {
		return nil, fmt.Errorf("configClient.GenerateService('%s', '%s', '%s'): %w", independent.id, independent.url, independent.Type, err)
	}
	generatedConfig.Manager.UrlFunc(clientConfig.Url)

	// Get all handlers and add them into the service
	for category, raw := range independent.Handlers {
		handler := raw.(base.Interface)
		generatedHandler, err := configClient.GenerateHandler(handler.Type(), category, false)
		if err != nil {
			return nil, fmt.Errorf("configClient.GenerateHandler('%s', '%s', internal: false): %w", handler.Type(), category, err)
		}

		handler.SetConfig(generatedHandler)

		generatedConfig.SetHandler(generatedHandler)
	}

	// Some handlers were generated and added into generated service config.
	// Notify the config engine to update the service.
	if len(independent.Handlers) > 0 {
		if err := configClient.SetService(generatedConfig); err != nil {
			return nil, fmt.Errorf("configClient.SetService('generated'): %w", err)
		}
	}

	return generatedConfig, nil
}

// lintConfig gets the configuration from the context and sets them in the service and handler.
func (independent *Service) lintConfig() error {
	configClient := independent.ctx.Config()

	returnedService, err := configClient.Service(independent.id)
	if err != nil {
		return fmt.Errorf("configClient.Service('%s', '%s', '%s'): %w", independent.id, independent.url, independent.Type, err)
	}
	returnedService.Manager.UrlFunc(clientConfig.Url)

	if returnedService.Url != independent.url {
		independent.url = returnedService.Url
	}
	if returnedService.Type != independent.Type {
		independent.Type = returnedService.Type
	}

	for category, raw := range independent.Handlers {
		handler := raw.(base.Interface)

		returnedHandler, err := returnedService.HandlerByCategory(category)
		if err != nil {
			generatedHandler, err := configClient.GenerateHandler(handler.Type(), category, false)
			if err != nil {
				return fmt.Errorf("configClient.GenerateHandler('%s', '%s', internal: false): %w", handler.Type(), category, err)
			}

			handler.SetConfig(generatedHandler)

			returnedService.SetHandler(generatedHandler)
			if err := configClient.SetService(returnedService); err != nil {
				return fmt.Errorf("configClient.SetService('returned'): %w", err)
			}
		} else {
			handler.SetConfig(returnedHandler)
		}
	}

	independent.config = returnedService

	return nil
}

// The setConfig sets the configuration of this service and handlers.
// If the configuration doesn't exist, generates the service and handler.
// The returned configuration from the context is linted into service and handler.
//
// Important node. This method doesn't set the proxies or extensions.
func (independent *Service) setConfig() error {
	configClient := independent.ctx.Config()

	// prepare the configuration
	exist, err := configClient.ServiceExist(independent.id)
	if err != nil {
		return fmt.Errorf("configClient.ServiceExist('%s'): %w", independent.id, err)
	}

	if !exist {
		generatedConfig, err := independent.generateConfig()
		if err != nil {
			return fmt.Errorf("generateConfig: %w", err)
		}
		independent.config = generatedConfig

		return nil
	}

	if err = independent.lintConfig(); err != nil {
		return fmt.Errorf("lintConfig: %w", err)
	}

	return nil
}

// The setProxyUnits gets the list of proxy chains for this service.
// Then, it creates a proxy units.
// todo if the extension is sending a ready command, then update the command list.
func (independent *Service) setProxyUnits() error {
	proxyClient := independent.ctx.ProxyClient()
	proxyChains, err := proxyClient.ProxyChainsByRuleUrl(independent.url)
	if err != nil {
		return fmt.Errorf("proxyClient.ProxyChainsByRuleUrl: %w", err)
	}

	// set the proxy destination units for each rule
	for _, proxyChain := range proxyChains {
		dest := proxyChain.Destination
		if dest.IsRoute() {
			units := independent.unitsByRouteRule(dest)
			if err := proxyClient.SetUnits(dest, units); err != nil {
				return fmt.Errorf("proxyClient.SetUnits: %w", err)
			}
		} else if dest.IsHandler() {
			units := independent.unitsByHandlerRule(dest)
			if err := proxyClient.SetUnits(dest, units); err != nil {
				return fmt.Errorf("proxyClient.SetUnits: %w", err)
			}
		} else if dest.IsService() {
			units := independent.unitsByServiceRule(dest)
			if err := proxyClient.SetUnits(dest, units); err != nil {
				return fmt.Errorf("proxyClient.SetUnits: %w", err)
			}
		}
	}

	return nil
}

// unitsByRouteRule returns the list of units for the route rule
func (independent *Service) unitsByRouteRule(rule *serviceConfig.Rule) []*serviceConfig.Unit {
	units := make([]*serviceConfig.Unit, 0, len(rule.Commands)*len(rule.Categories))

	if len(independent.Handlers) == 0 {
		return units
	}

	for _, raw := range independent.Handlers {
		handlerInterface := raw.(base.Interface)
		hConfig := handlerInterface.Config()

		if !slices.Contains(rule.Categories, hConfig.Category) {
			continue
		}

		for _, command := range rule.Commands {
			if slices.Contains(rule.ExcludedCommands, command) {
				continue
			}

			if !handlerInterface.IsRouteExist(command) {
				continue
			}

			unit := &serviceConfig.Unit{
				ServiceId: independent.id,
				HandlerId: hConfig.Id,
				Command:   command,
			}

			units = append(units, unit)
		}
	}

	return units
}

// unitsByHandlerRule returns the list of units for the handler rule
func (independent *Service) unitsByHandlerRule(rule *serviceConfig.Rule) []*serviceConfig.Unit {
	units := make([]*serviceConfig.Unit, 0, len(rule.Categories))

	for _, raw := range independent.Handlers {
		handlerInterface := raw.(base.Interface)
		hConfig := handlerInterface.Config()

		if !slices.Contains(rule.Categories, hConfig.Category) {
			continue
		}

		commands := handlerInterface.RouteCommands()

		for _, command := range commands {
			if slices.Contains(rule.ExcludedCommands, command) {
				continue
			}

			unit := &serviceConfig.Unit{
				ServiceId: independent.id,
				HandlerId: hConfig.Id,
				Command:   command,
			}

			units = append(units, unit)
		}
	}

	return units
}

// unitsByServiceRule returns the list of units for the service rule
func (independent *Service) unitsByServiceRule(rule *serviceConfig.Rule) []*serviceConfig.Unit {
	units := make([]*serviceConfig.Unit, 0, len(rule.Categories))

	for _, raw := range independent.Handlers {
		handlerInterface := raw.(base.Interface)
		hConfig := handlerInterface.Config()

		commands := handlerInterface.RouteCommands()

		for _, command := range commands {
			if slices.Contains(rule.ExcludedCommands, command) {
				continue
			}

			unit := &serviceConfig.Unit{
				ServiceId: independent.id,
				HandlerId: hConfig.Id,
				Command:   command,
			}

			units = append(units, unit)
		}
	}

	return units
}

// newManager creates a manager.Manager and assigns it to manager, otherwise manager is nil.
//
// The manager.Manager depends on config set by setConfig.
//
// The manager.Manager depends on Logger, set automatically.
//
// This function lints manager.Manager with ctx.
func (independent *Service) newManager() error {
	if independent.config == nil {
		return fmt.Errorf("independent.config is nill")
	}
	m, err := manager.New(independent.ctx, &independent.blocker, independent.config.Manager)
	if err != nil {
		return fmt.Errorf("manager.New: %w", err)
	}
	err = m.SetLogger(independent.Logger)
	if err != nil {
		return fmt.Errorf("manager.SetLogger: %w", err)
	}
	independent.manager = m

	return nil
}

// setHandlerClient creates a handler manager clients and sets them into the service manager.
func (independent *Service) setHandlerClient(c base.Interface) error {
	handlerClient, err := manager_client.New(c.Config())
	if err != nil {
		return fmt.Errorf("manager_client.New('%s'): %w", c.Config().Category, err)
	}
	independent.manager.SetHandlerManagers([]manager_client.Interface{handlerClient})

	return nil
}

// startHandler sets the log into the handler which is prepared already.
// then, starts it.
func (independent *Service) startHandler(handler base.Interface) error {
	if err := handler.SetLogger(independent.Logger); err != nil {
		return fmt.Errorf("handler(id: '%s').SetLogger: %w", handler.Config().Id, err)
	}

	if err := handler.Start(); err != nil {
		return fmt.Errorf("handler(category: '%s').Start: %w", handler.Config().Category, err)
	}

	return nil
}

// Start the service.
//
// Requires at least one handler.
func (independent *Service) Start() (*sync.WaitGroup, error) {
	var err error
	var startedHandlers []string

	if len(independent.Handlers) == 0 {
		err = fmt.Errorf("no Handlers. call service.SetHandler")
		goto errOccurred
	}

	if err = independent.setConfig(); err != nil {
		err = fmt.Errorf("setConfig: %w", err)
		goto errOccurred
	}

	independent.ctx.SetService(independent.id, independent.url)
	if err = independent.ctx.StartDepManager(); err != nil {
		err = fmt.Errorf("ctx.StartDepManager: %w", err)
		goto errOccurred
	}
	if err = independent.ctx.StartProxyHandler(); err != nil {
		err = fmt.Errorf("ctx.StartProxyHandler: %w", err)
		goto errOccurred
	}

	if err = independent.newManager(); err != nil {
		err = fmt.Errorf("newManager: %w", err)
		goto errOccurred
	}

	// get the proxies from the proxy chain for this service.
	if err = independent.setProxyUnits(); err != nil {
		err = fmt.Errorf("independent.setProxyUnits: %w", err)
		goto errOccurred
	}

	for category, raw := range independent.Handlers {
		handler := raw.(base.Interface)
		if err = independent.setHandlerClient(handler); err != nil {
			err = fmt.Errorf("setHandlerClient('%s'): %w", category, err)
			goto errOccurred
		}

		if err = independent.startHandler(handler); err != nil {
			err = fmt.Errorf("startHandler: %w", err)
			goto errOccurred
		}

		startedHandlers = append(startedHandlers, category)
	}

	// todo prepare the extensions by calling them in the context.
	// todo prepare the extensions by setting them into the independent.manager.
	// prepare the proxies by setting them into the independent.manager.
	// prepare the extensions by calling them in the context.
	// prepare the extensions by setting them into the independent.manager.

	if err = independent.manager.Start(); err != nil {
		err = fmt.Errorf("service.manager.Start: %w", err)
		goto errOccurred
	}

	if err = independent.ctx.ProxyClient().StartLastProxies(); err != nil {
		err = fmt.Errorf("ctx.ProxyClient.StartLastProxies: %w", err)
		goto errOccurred
	}

	//err = independent.Context.ServiceReady(independent.Logger)
	//if err != nil {
	//	goto errOccurred
	//}

errOccurred:
	if err != nil {
		closeErr := independent.ctx.Close()
		if closeErr != nil {
			err = fmt.Errorf("%v: ctx.Close: %w", err, closeErr)
		}

		for _, category := range startedHandlers {
			handler := independent.Handlers[category].(base.Interface)
			handlerClient, newErr := manager_client.New(handler.Config())
			if newErr != nil {
				err = fmt.Errorf("%v: manager_client.New('%s'): %w", err, category, newErr)
			} else {
				if closeErr = handlerClient.Close(); closeErr != nil {
					err = fmt.Errorf("%v: handlerClient('%s').Close: %w", err, category, closeErr)
				}
			}
		}

		if independent.manager.Running() {
			closeErr = independent.manager.Close()
			if closeErr != nil {
				err = fmt.Errorf("%v: manager.Close: %w", err, closeErr)
			}
		}
	}

	if err == nil {
		independent.blocker = &sync.WaitGroup{}
		independent.blocker.Add(1)
	}

	return independent.blocker, err
}

//func (independent *Service) prepareExtensionConfiguration(dep *dev.Dep) error {
//	err := dep.Prepare(independent.Logger)
//	if err != nil {
//		return fmt.Errorf("dev.Prepare(%s): %w", dep.Url(), err)
//	}
//
//	err = dep.PrepareConfig(independent.Logger)
//	if err != nil {
//		return fmt.Errorf("dev.PrepareConfig on %s: %w", dep.Url(), err)
//	}
//
//	//depConfig, err := dep.GetServiceConfig()
//	//converted, err := converter.ServiceToExtension(depConfig)
//	//if err != nil {
//	//	return fmt.Errorf("config.ServiceToExtension: %w", err)
//	//}
//	//
//	//extensionConfiguration := independent.config.GetExtension(dep.Url())
//	//if extensionConfiguration == nil {
//	//	independent.config.SetExtension(&converted)
//	//} else {
//	//	if strings.Compare(extensionConfiguration.Url, converted.Url) != 0 {
//	//		return fmt.Errorf("the extension url in your '%s' config not matches to '%s' in the dependency", extensionConfiguration.Url, converted.Url)
//	//	}
//	//	if extensionConfiguration.Port != converted.Port {
//	//		independent.Logger.Warn("dependency port not matches to the extension port. Overwriting the source", "port", extensionConfiguration.Port, "dependency port", converted.Port)
//	//
//	//		main, _ := depConfig.GetFirstController()
//	//		main.Instances[0].Port = extensionConfiguration.Port
//	//
//	//depConfig.SetController(main)
//	//
//	//err = dep.SetServiceConfig(depConfig)
//	//if err != nil {
//	//	return fmt.Errorf("failed to update port in dependency extension: '%s': %w", dep.Url(), err)
//	//}
//	//}
//	//}
//
//	return nil
//}
