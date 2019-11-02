package pwauth

import (
	"errors"
	"fmt"
	"github.com/msteinert/pam"
)

func auth(username, password string) error {
	t, err := pam.StartFunc("rttys", username, func(s pam.Style, msg string) (string, error) {
		switch s {
		case pam.PromptEchoOff:
			return password, nil
		case pam.PromptEchoOn:
			return password, nil
		case pam.ErrorMsg:
			fmt.Print(msg)
			return "", nil
		case pam.TextInfo:
			fmt.Println(msg)
			return "", nil
		}
		return "", errors.New("Unrecognized message style")
	})
	if err != nil {
		return err
	}

	err = t.Authenticate(0)
	if err != nil {
		return err
	}

	return nil
}
