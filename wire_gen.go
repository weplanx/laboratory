// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"lab-api/app"
	"lab-api/app/api"
	"lab-api/app/api/dev"
	"lab-api/app/api/index"
	"lab-api/app/system"
	"lab-api/app/system/admin"
	index2 "lab-api/app/system/index"
	"lab-api/app/system/resource"
	"lab-api/common"
)

// Injectors from wire.go:

func App(set *common.Set) (*app.App, error) {
	engine := common.HttpServer(set)
	db, err := common.InitializeDatabase(set)
	if err != nil {
		return nil, err
	}
	client, err := common.InitializeRedis(set)
	if err != nil {
		return nil, err
	}
	cookie := common.InitializeCookie(set)
	authx := common.InitializeAuthx(set)
	cipher, err := common.InitializeCipher(set)
	if err != nil {
		return nil, err
	}
	dependency := &common.Dependency{
		Set:    set,
		Db:     db,
		Redis:  client,
		Cookie: cookie,
		Authx:  authx,
		Cipher: cipher,
	}
	service := index.NewService(dependency)
	controllerInject := &index.ControllerInject{
		Service: service,
	}
	controller := index.NewController(dependency, controllerInject)
	devService := dev.NewService(dependency)
	devControllerInject := &dev.ControllerInject{
		Service: devService,
	}
	devController := dev.NewController(dependency, devControllerInject)
	inject := &api.Inject{
		Index: controller,
		Dev:   devController,
	}
	routes := api.NewRoutes(engine, inject)
	indexService := index2.NewService(dependency)
	resourceService := resource.NewService(dependency)
	adminService := admin.NewService(dependency)
	indexControllerInject := &index2.ControllerInject{
		Service:         indexService,
		ResourceService: resourceService,
		AdminService:    adminService,
	}
	indexController := index2.NewController(dependency, indexControllerInject, authx)
	resourceControllerInject := &resource.ControllerInject{
		Service: resourceService,
	}
	resourceController := resource.NewController(dependency, resourceControllerInject)
	systemInject := &system.Inject{
		Index:    indexController,
		Resource: resourceController,
	}
	systemRoutes := system.NewRoutes(engine, dependency, systemInject)
	appApp := app.NewApp(engine, routes, systemRoutes)
	return appApp, nil
}
