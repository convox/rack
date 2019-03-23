package api

import (
	"fmt"

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

func (s *Server) ProcessExecValidate(c *stdapi.Context) error {
	if _, err := s.Provider.AppGet(c.Var("app")); err != nil {
		return err
	}

	return nil
}

func (s *Server) ProcessRunValidate(c *stdapi.Context) error {
	if _, err := s.Provider.AppGet(c.Var("app")); err != nil {
		return err
	}

	return nil
}

func (s *Server) ReleasePromoteValidate(c *stdapi.Context) error {
	a, err := s.Provider.AppGet(c.Var("app"))
	if err != nil {
		return err
	}

	if c.Form("force") != "true" && a.Status != "running" {
		return stdapi.Errorf(403, "app is currently updating")
	}

	if a.Release != "" {
		or, err := s.Provider.ReleaseGet(c.Var("app"), a.Release)
		if err != nil {
			return err
		}

		nr, err := s.Provider.ReleaseGet(c.Var("app"), c.Var("id"))
		if err != nil {
			return err
		}

		if nr.Created.Before(or.Created) {
			return fmt.Errorf("can not promote an older release, try rollback")
		}
	}

	return nil
}
