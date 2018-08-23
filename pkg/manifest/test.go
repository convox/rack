package manifest

import "io"

type TestOptions struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (m *Manifest) Test(ns string, opts TestOptions) error {
	for _, s := range m.Services {
		if s.Test != "" {
			err := s.run(ns, s.Test, RunOptions{
				Stdout: opts.Stdout,
				Stderr: opts.Stderr,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}
