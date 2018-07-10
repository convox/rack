package stdcli

import (
	"fmt"
)

type Validator func(c *Context) error

func Args(num int) Validator {
	return func(c *Context) error {
		if len(c.Args) != num {
			return fmt.Errorf("%d %s required", num, plural("arg", num))
		}
		return nil
	}
}

func ArgsBetween(min, max int) Validator {
	return func(c *Context) error {
		if err := ArgsMin(min)(c); err != nil {
			return err
		}
		if err := ArgsMax(max)(c); err != nil {
			return err
		}
		return nil
	}
}

func ArgsMin(min int) Validator {
	return func(c *Context) error {
		if len(c.Args) < min {
			return fmt.Errorf("at least %d %s required", min, plural("arg", min))
		}
		return nil
	}
}

func ArgsMax(max int) Validator {
	return func(c *Context) error {
		if len(c.Args) > max {
			return fmt.Errorf("no more than %d %s expected", max, plural("arg", max))
		}
		return nil
	}
}

func plural(noun string, num int) string {
	if num == 1 {
		return noun
	}

	return fmt.Sprintf("%ss", noun)
}
