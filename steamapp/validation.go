package steamapp

import "fmt"

func ValidateAppID(appID int) error {
	if appID >= 10 && appID%10 == 0 {
		return nil
	}

	return fmt.Errorf("invalid Steamapp ID: must be greater than and divisible by 10: %d", appID)
}
