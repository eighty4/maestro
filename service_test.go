package main

import "testing"

func TestSortServiceDeps_WithTwoServices(t *testing.T) {
	var services = []*ServiceConfig{
		{Name: "postgres"},
		{Name: "service", DependsOn: []string{"postgres"}},
	}
	result, err := sortServiceDeps(services)
	if err != nil {
		t.Error(err)
	}
	if result[0] != "postgres" {
		t.Error("first should be postgres")
	}
	if result[1] != "service" {
		t.Error("last should be service")
	}
}

func TestSortServiceDeps_With(t *testing.T) {
	var services = []*ServiceConfig{
		{Name: "service1", DependsOn: []string{"postgres"}},
		{Name: "postgres"},
		{Name: "service2", DependsOn: []string{"postgres", "service1"}},
	}
	result, err := sortServiceDeps(services)
	if err != nil {
		t.Error(err)
	}
	if result[0] != "postgres" {
		t.Error("first should be postgres")
	}
	if result[1] != "service1" {
		t.Error("second should be service1")
	}
	if result[2] != "service2" {
		t.Error("last should be service2")
	}
}

func TestSortServiceDeps_ErrorsWithCircularDep(t *testing.T) {
	var services = []*ServiceConfig{
		{Name: "service1", DependsOn: []string{"service2"}},
		{Name: "service2", DependsOn: []string{"service1"}},
	}
	result, err := sortServiceDeps(services)
	if err == nil {
		t.Error("error not present")
	}
	if !IsCircularDepError(err) {
		t.Error(err)
	}
	if result != nil {
		t.Error("expects nil result")
	}
}
