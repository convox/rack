package api

import (
	"github.com/convox/stdapi"
)

func (s *Server) AppCancelValidate(c *stdapi.Context) error {
	a, err := s.Provider.AppGet(c.Var("name"))
	if err != nil {
		return err
	}

	if a.Status != "updating" {
		return stdapi.Errorf(403, "app is not updating")
	}

	return nil
}

func (s *Server) ReleasePromoteValidate(c *stdapi.Context) error {
	a, err := s.Provider.AppGet(c.Var("app"))
	if err != nil {
		return err
	}

	if a.Status != "running" {
		return stdapi.Errorf(403, "app is currently updating")
	}

	return nil
}
